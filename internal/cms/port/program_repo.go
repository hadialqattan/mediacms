package port

import (
	"context"

	"thmanyah.com/content-platform/internal/cms/repository/sqlc"
	"thmanyah.com/content-platform/internal/shared/domain"
)

type ProgramRepo interface {
	Create(ctx context.Context, params sqlc.CreateProgramParams) (*domain.Program, error)
	GetByID(ctx context.Context, id string) (*domain.Program, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Program, error)
	List(ctx context.Context) ([]*domain.Program, error)
	Update(ctx context.Context, id string, params sqlc.UpdateProgramParams) (*domain.Program, error)
	Publish(ctx context.Context, id, publishedBy string) (*domain.Program, error)
	Delete(ctx context.Context, id, deletedBy string) error
	AssignCategories(ctx context.Context, programID string, categoryIDs []string) error
	GetCategories(ctx context.Context, programID string) ([]domain.Category, error)
}
