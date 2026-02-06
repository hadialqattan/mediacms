package port

import (
	"context"

	"thmanyah.com/content-platform/internal/shared/domain"
)

type OutboxRepo interface {
	GetPending(ctx context.Context) ([]*domain.OutboxEvent, error)
	MarkEnqueued(ctx context.Context, id string) error
}
