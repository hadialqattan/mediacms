package repository

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"

	"thmanyah.com/content-platform/internal/cms/port"
	"thmanyah.com/content-platform/internal/cms/repository/sqlc"
	"thmanyah.com/content-platform/internal/shared/domain"
)

type outboxRepo struct {
	queries *sqlc.Queries
}

func NewOutboxRepo(pool *pgxpool.Pool) port.OutboxRepo {
	return &outboxRepo{
		queries: sqlc.New(pool),
	}
}

func (r *outboxRepo) Create(ctx context.Context, params sqlc.CreateOutboxEventParams) (*domain.OutboxEvent, error) {
	event, err := r.queries.CreateOutboxEvent(ctx, params)
	if err != nil {
		return nil, err
	}
	return r.domainOutboxEvent(event), nil
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
