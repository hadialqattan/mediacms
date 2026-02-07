package port

import (
	"context"

	"thmanyah.com/content-platform/internal/shared/domain"
)

type SearchIndex interface {
	UpsertProgram(ctx context.Context, program domain.Program) error
	DeleteProgram(ctx context.Context, programID string) error
}
