package port

import (
	"context"

	"thmanyah.com/content-platform/internal/shared/domain"
)

type ProgramReader interface {
	GetByID(ctx context.Context, id string) (*domain.Program, error)
	GetCategories(ctx context.Context, programID string) ([]domain.Category, error)
}

type SearchIndex interface {
	UpsertProgram(ctx context.Context, program domain.Program) error
	DeleteProgram(ctx context.Context, programID string) error
}
