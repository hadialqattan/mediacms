package searchindexer

import (
	"context"
	"encoding/json"
	"log"

	"github.com/hibiken/asynq"

	"thmanyah.com/content-platform/internal/searchindexer/port"
	"thmanyah.com/content-platform/internal/shared/domain"
)

type Worker struct {
	programReader port.ProgramReader
	searchIndex   port.SearchIndex
}

func NewWorker(
	programReader port.ProgramReader,
	searchIndex port.SearchIndex,
) *Worker {
	return &Worker{
		programReader: programReader,
		searchIndex:   searchIndex,
	}
}

func (w *Worker) HandleProgramUpsert(ctx context.Context, t *asynq.Task) error {
	var payload map[string]interface{}
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return err
	}

	programID, ok := payload["program_id"].(string)
	if !ok {
		log.Printf("Invalid payload for program.upsert: missing program_id")
		return nil
	}

	program, err := w.programReader.GetByID(ctx, programID)
	if err != nil {
		log.Printf("Failed to get program %s: %v", programID, err)
		return err
	}

	if !program.IsPublished() {
		log.Printf("Program %s is not published, skipping indexing", programID)
		return nil
	}

	if program.IsDeleted() {
		log.Printf("Program %s is deleted, skipping indexing", programID)
		return nil
	}

	categories, err := w.programReader.GetCategories(ctx, programID)
	if err != nil {
		log.Printf("Failed to get categories for program %s: %v", programID, err)
	} else {
		program.Categories = categories
	}

	if err := w.searchIndex.UpsertProgram(ctx, *program); err != nil {
		log.Printf("Failed to upsert program %s to search index: %v", programID, err)
		return err
	}

	log.Printf("Successfully indexed program %s", programID)
	return nil
}

func (w *Worker) HandleProgramDelete(ctx context.Context, t *asynq.Task) error {
	var payload map[string]interface{}
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return err
	}

	programID, ok := payload["program_id"].(string)
	if !ok {
		log.Printf("Invalid payload for program.delete: missing program_id")
		return nil
	}

	if err := w.searchIndex.DeleteProgram(ctx, programID); err != nil {
		log.Printf("Failed to delete program %s from search index: %v", programID, err)
		return err
	}

	log.Printf("Successfully deleted program %s from search index", programID)
	return nil
}

func NewWorkerAndMux(worker *Worker) (*asynq.Server, *asynq.ServeMux) {
	mux := asynq.NewServeMux()

	mux.HandleFunc(string(domain.OutboxEventTypeProgramUpsert), worker.HandleProgramUpsert)
	mux.HandleFunc(string(domain.OutboxEventTypeProgramDelete), worker.HandleProgramDelete)

	server := asynq.NewServer(
		asynq.RedisClientOpt{Addr: "localhost:6379"},
		asynq.Config{
			Concurrency: 1,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				log.Printf("Task %s failed: %v", task.Type, err)
			}),
		},
	)

	return server, mux
}
