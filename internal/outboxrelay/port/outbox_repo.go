package port

import (
	"context"

	"github.com/jackc/pgx/v5"

	"thmanyah.com/content-platform/internal/shared/domain"
)

type OutboxRepo interface {
	GetPending(ctx context.Context) ([]*domain.OutboxEvent, error)
	MarkEnqueued(ctx context.Context, id string) error
	WithTx(tx pgx.Tx) OutboxRepo
}
