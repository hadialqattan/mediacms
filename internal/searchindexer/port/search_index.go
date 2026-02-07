package port

import (
	"context"

	"github.com/hadialqattan/mediacms/internal/shared/domain"
)

type SearchIndex interface {
	UpsertProgram(ctx context.Context, program domain.Program) error
	DeleteProgram(ctx context.Context, programID string) error
}
