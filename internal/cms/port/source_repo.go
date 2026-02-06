package port

import (
	"context"

	"thmanyah.com/content-platform/internal/cms/repository/sqlc"
	"thmanyah.com/content-platform/internal/shared/domain"
)

type SourceRepo interface {
	Create(ctx context.Context, params sqlc.CreateSourceParams) (*domain.Source, error)
	GetByID(ctx context.Context, id string) (*domain.Source, error)
}
