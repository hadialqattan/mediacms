package repository

import (
	"context"
	"fmt"

	"github.com/typesense/typesense-go/typesense"
	"github.com/typesense/typesense-go/typesense/api"

	"thmanyah.com/content-platform/internal/discovery/port"
	"thmanyah.com/content-platform/internal/shared/domain"
)

const collectionName = "programs"

type searchIndex struct {
	client *typesense.Client
}

func NewSearchIndex(client *typesense.Client) port.SearchIndex {
	return &searchIndex{client: client}
}

func (s *searchIndex) UpsertProgram(ctx context.Context, program domain.Program) error {
	if !program.IsPublished() {
		return nil
	}

	categories := make([]string, len(program.Categories))
	for i, c := range program.Categories {
		categories[i] = c.Name
	}

	var publishedAt int64
	if program.PublishedAt != nil {
		publishedAt = program.PublishedAt.Unix()
	}

	document := map[string]interface{}{
		"id":           program.ID,
		"slug":         program.Slug,
		"title":        program.Title,
		"description":  program.Description,
		"type":         string(program.Type),
		"language":     string(program.Language),
		"duration_ms":  program.DurationMs,
		"categories":   categories,
		"published_at": publishedAt,
	}

	_, err := s.client.Collection(collectionName).Documents().Upsert(ctx, document)
	if err != nil {
		return fmt.Errorf("upserting document: %w", err)
	}
	return nil
}

func (s *searchIndex) DeleteProgram(ctx context.Context, programID string) error {
	_, err := s.client.Collection(collectionName).Document(programID).Delete(ctx)
	if err != nil {
		return fmt.Errorf("deleting document: %w", err)
	}
	return nil
}

func (s *searchIndex) SearchPrograms(ctx context.Context, params port.SearchParams) (*port.SearchResult, error) {
	page := params.Page
	perPage := params.PerPage

	searchParams := &api.SearchCollectionParams{
		Q:       params.Query,
		QueryBy: "title,description",
		Page:    &page,
		PerPage: &perPage,
	}

	if params.ProgramType != nil {
		filter := fmt.Sprintf("type:=%s", *params.ProgramType)
		searchParams.FilterBy = &filter
	}
	if params.Language != nil {
		existingFilter := ""
		if searchParams.FilterBy != nil {
			existingFilter = *searchParams.FilterBy + " && "
		}
		filter := existingFilter + fmt.Sprintf("language:=%s", *params.Language)
		searchParams.FilterBy = &filter
	}
	if len(params.Categories) > 0 {
		existingFilter := ""
		if searchParams.FilterBy != nil {
			existingFilter = *searchParams.FilterBy + " && "
		}
		categoryFilter := ""
		for i, cat := range params.Categories {
			if i > 0 {
				categoryFilter += " || "
			}
			categoryFilter += fmt.Sprintf("categories:=%s", cat)
		}
		filter := existingFilter + fmt.Sprintf("(%s)", categoryFilter)
		searchParams.FilterBy = &filter
	}

	result, err := s.client.Collection(collectionName).Documents().Search(ctx, searchParams)
	if err != nil {
		return nil, fmt.Errorf("searching documents: %w", err)
	}

	programs := make([]domain.Program, 0)
	if result.Hits != nil {
		for _, hit := range *result.Hits {
			if hit.Document != nil {
				doc := *hit.Document
				program := domain.Program{
					ID:          getString(doc, "id"),
					Slug:        getString(doc, "slug"),
					Title:       getString(doc, "title"),
					Description: getString(doc, "description"),
					Type:        domain.ProgramType(getString(doc, "type")),
					Language:    domain.ProgramLanguage(getString(doc, "language")),
					DurationMs:  getInt(doc, "duration_ms"),
					Categories:  []domain.Category{},
				}
				programs = append(programs, program)
			}
		}
	}

	totalFound := 0
	if result.Found != nil {
		totalFound = int(*result.Found)
	}

	return &port.SearchResult{
		Programs:   programs,
		TotalFound: totalFound,
		Page:       params.Page,
		PerPage:    params.PerPage,
	}, nil
}

func (s *searchIndex) GetProgram(ctx context.Context, programID string) (*domain.Program, error) {
	doc, err := s.client.Collection(collectionName).Document(programID).Retrieve(ctx)
	if err != nil {
		return nil, fmt.Errorf("retrieving document: %w", err)
	}

	program := &domain.Program{
		ID:          getString(doc, "id"),
		Slug:        getString(doc, "slug"),
		Title:       getString(doc, "title"),
		Description: getString(doc, "description"),
		Type:        domain.ProgramType(getString(doc, "type")),
		Language:    domain.ProgramLanguage(getString(doc, "language")),
		DurationMs:  getInt(doc, "duration_ms"),
		Categories:  []domain.Category{},
	}

	return program, nil
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return int(val)
		case int:
			return val
		case int32:
			return int(val)
		case int64:
			return int(val)
		}
	}
	return 0
}
