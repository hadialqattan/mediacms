package port

import (
	"context"

	"thmanyah.com/content-platform/internal/cms/repository/sqlc"
	"thmanyah.com/content-platform/internal/shared/domain"
)

type CategoryRepo interface {
	Create(ctx context.Context, params sqlc.CreateCategoryParams) (*domain.Category, error)
	GetByID(ctx context.Context, id string) (*domain.Category, error)
	GetByName(ctx context.Context, name string) (*domain.Category, error)
	List(ctx context.Context) ([]*domain.Category, error)
}
