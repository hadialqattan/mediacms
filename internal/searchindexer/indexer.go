package searchindexer

import (
	"context"

	"github.com/hibiken/asynq"
)

type Indexer struct {
	worker *Worker
}

func NewIndexer(worker *Worker) *Indexer {
	return &Indexer{worker: worker}
}

func (i *Indexer) ProcessTask(ctx context.Context, eventType string, task *asynq.Task) error {
	switch eventType {
	case "program.upsert":
		return i.worker.HandleProgramUpsert(ctx, task)
	case "program.delete":
		return i.worker.HandleProgramDelete(ctx, task)
	default:
		return nil
	}
}
