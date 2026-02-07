package repository

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"thmanyah.com/content-platform/internal/outboxrelay/port"
	"thmanyah.com/content-platform/internal/outboxrelay/repository/sqlc"
	"thmanyah.com/content-platform/internal/shared/domain"
)

type outboxRepo struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewOutboxRepo(pool *pgxpool.Pool) port.OutboxRepo {
	return &outboxRepo{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

func (r *outboxRepo) WithTx(tx pgx.Tx) port.OutboxRepo {
	return &outboxRepo{
		pool:    r.pool,
		queries: r.queries.WithTx(tx),
	}
}

func (r *outboxRepo) GetPending(ctx context.Context) ([]*domain.OutboxEvent, error) {
	events, err := r.queries.GetPendingOutboxEvents(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*domain.OutboxEvent, len(events))
	for i, e := range events {
		result[i] = r.domainOutboxEvent(e)
	}
	return result, nil
}

func (r *outboxRepo) MarkEnqueued(ctx context.Context, id string) error {
	uid, err := uuid.Parse(id)
	if err != nil {
		return err
	}

	return r.queries.MarkOutboxEventEnqueued(ctx, pgtype.UUID{Bytes: uid, Valid: true})
}

func (r *outboxRepo) domainOutboxEvent(e sqlc.OutboxEvent) *domain.OutboxEvent {
	var payload map[string]interface{}
	json.Unmarshal(e.Payload, &payload)

	event := &domain.OutboxEvent{
		ID:        e.ID.String(),
		Type:      domain.OutboxEventType(e.Type),
		Payload:   payload,
		Enqueued:  e.Enqueued,
		CreatedAt: e.CreatedAt.Time,
	}

	if e.ProgramID.Valid {
		programID := e.ProgramID.String()
		event.ProgramID = &programID
	}

	return event
}
