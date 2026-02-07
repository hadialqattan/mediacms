package searchindexer

import (
	"context"

	"github.com/hadialqattan/mediacms/internal/shared/domain"
	"github.com/hibiken/asynq"
)

type Indexer struct {
	worker *Worker
}

func NewIndexer(worker *Worker) *Indexer {
	return &Indexer{worker: worker}
}

func (i *Indexer) ProcessTask(ctx context.Context, eventType domain.OutboxEventType, task *asynq.Task) error {
	switch eventType {
	case domain.OutboxEventTypeProgramUpsert:
		return i.worker.HandleProgramUpsert(ctx, task)
	case domain.OutboxEventTypeProgramDelete:
		return i.worker.HandleProgramDelete(ctx, task)
	default:
		return nil
	}
}
