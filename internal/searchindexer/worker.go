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
	searchIndex port.SearchIndex
}

func NewWorker(
	searchIndex port.SearchIndex,
) *Worker {
	return &Worker{
		searchIndex: searchIndex,
	}
}

func (w *Worker) HandleProgramUpsert(ctx context.Context, t *asynq.Task) error {
	var program domain.Program
	if err := json.Unmarshal(t.Payload(), &program); err != nil {
		log.Printf("Failed to unmarshal program payload: %v", err)
		return err
	}

	if !program.IsPublished() {
		log.Printf("Program %s is not published, skipping indexing", program.ID)
		return nil
	}

	if program.IsDeleted() {
		log.Printf("Program %s is deleted, skipping indexing", program.ID)
		return nil
	}

	if err := w.searchIndex.UpsertProgram(ctx, program); err != nil {
		log.Printf("Failed to upsert program %s to search index: %v", program.ID, err)
		return err
	}

	log.Printf("Successfully indexed program %s", program.ID)
	return nil
}

func (w *Worker) HandleProgramDelete(ctx context.Context, t *asynq.Task) error {
	var program domain.Program
	if err := json.Unmarshal(t.Payload(), &program); err != nil {
		log.Printf("Failed to unmarshal program payload: %v", err)
		return err
	}

	if err := w.searchIndex.DeleteProgram(ctx, program.ID); err != nil {
		log.Printf("Failed to delete program %s from search index: %v", program.ID, err)
		return err
	}

	log.Printf("Successfully deleted program %s from search index", program.ID)
	return nil
}

func NewWorkerAndMux(worker *Worker, redisAddr string) (*asynq.Server, *asynq.ServeMux) {
	mux := asynq.NewServeMux()

	mux.HandleFunc(string(domain.OutboxEventTypeProgramUpsert), worker.HandleProgramUpsert)
	mux.HandleFunc(string(domain.OutboxEventTypeProgramDelete), worker.HandleProgramDelete)

	server := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			Concurrency: 1,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				log.Printf("Task %s failed: %v", task.Type(), err)
			}),
		},
	)

	return server, mux
}
