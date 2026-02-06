package port

import (
	"context"

	"thmanyah.com/content-platform/internal/cms/repository/sqlc"
	"thmanyah.com/content-platform/internal/shared/domain"
)

type OutboxRepo interface {
	Create(ctx context.Context, params sqlc.CreateOutboxEventParams) (*domain.OutboxEvent, error)
}
