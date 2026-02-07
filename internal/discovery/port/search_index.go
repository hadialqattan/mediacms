package port

import (
	"context"

	"github.com/hadialqattan/mediacms/internal/shared/domain"
)

type SearchIndex interface {
	UpsertProgram(ctx context.Context, program domain.Program) error
	DeleteProgram(ctx context.Context, programID string) error
	SearchPrograms(ctx context.Context, params SearchParams) (*SearchResult, error)
	GetProgram(ctx context.Context, programID string) (*domain.Program, error)
}

type SearchParams struct {
	Query       string
	ProgramType *string
	Language    *string
	Categories  []string
	Page        int
	PerPage     int
}

type SearchResult struct {
	Programs   []domain.Program `json:"programs"`
	TotalFound int              `json:"total_found"`
	Page       int              `json:"page"`
	PerPage    int              `json:"per_page"`
}
