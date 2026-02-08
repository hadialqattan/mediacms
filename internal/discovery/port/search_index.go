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
	GetRecentPrograms(ctx context.Context, params RecentParams) (*RecentResult, error)
	GetFacets(ctx context.Context) (*FacetsResult, error)
}

type SearchParams struct {
	Query       string
	ProgramType *string
	Language    *string
	Tags        []string
	Page        int
	PerPage     int
	Sort        *string
}

type SearchResult struct {
	Results    []ProgramResult `json:"results"`
	Facets     *Facets         `json:"facets,omitempty"`
	Pagination Pagination      `json:"pagination"`
}

type ProgramResult struct {
	ID          string           `json:"id"`
	Slug        string           `json:"slug"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Type        domain.ProgramType `json:"type"`
	Language    domain.ProgramLanguage `json:"language"`
	DurationMs  int              `json:"duration_ms"`
	Tags        []string         `json:"tags"`
	PublishedAt int64            `json:"published_at"`
	CreatedAt   int64            `json:"created_at"`
}

type RecentParams struct {
	ProgramType *string
	Language    *string
	Page        int
	PerPage     int
}

type RecentResult struct {
	Results    []ProgramResult `json:"results"`
	Pagination Pagination      `json:"pagination"`
}

type FacetsResult struct {
	Facets Facets `json:"facets"`
}

type Facets struct {
	Type     map[string]int  `json:"type"`
	Language map[string]int  `json:"language"`
	Tags     map[string]int  `json:"tags"`
}

type Pagination struct {
	Total      int `json:"total"`
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
}
