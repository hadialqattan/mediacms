package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"thmanyah.com/content-platform/internal/outboxrelay/port"
	"thmanyah.com/content-platform/internal/shared/domain"
)

type outboxRepo struct {
	queries interface {
		GetPendingOutboxEvents(ctx context.Context) ([]interface{}, error)
		MarkOutboxEventEnqueued(ctx context.Context, id uuid.UUID) error
	}
}

func NewOutboxRepo(pool *pgxpool.Pool) port.OutboxRepo {
	return &outboxRepo{}
}

func (r *outboxRepo) GetPending(ctx context.Context) ([]*domain.OutboxEvent, error) {
	return nil, nil
}

func (r *outboxRepo) MarkEnqueued(ctx context.Context, id string) error {
	_, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	// TODO: After SQLC generation, implement: r.queries.MarkOutboxEventEnqueued(ctx, uid)
	return nil
}
