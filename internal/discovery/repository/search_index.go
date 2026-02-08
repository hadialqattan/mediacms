package repository

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/typesense/typesense-go/typesense"
	"github.com/typesense/typesense-go/typesense/api"

	"github.com/hadialqattan/mediacms/internal/discovery/port"
	"github.com/hadialqattan/mediacms/internal/shared/domain"
)

const collectionName = "programs"

type SearchIndex struct {
	client *typesense.Client
}

func NewSearchIndex(client *typesense.Client) port.SearchIndex {
	return &SearchIndex{client: client}
}

func (s *SearchIndex) UpsertProgram(ctx context.Context, program domain.Program) error {
	if !program.IsPublished() {
		return nil
	}

	document := map[string]interface{}{
		"id":           program.ID,
		"slug":         program.Slug,
		"title":        program.Title,
		"description":  program.Description,
		"type":         string(program.Type),
		"language":     string(program.Language),
		"tags":         program.Tags,
		"duration_ms":  program.DurationMs,
		"published_at": program.PublishedAt.Unix(),
		"created_at":   program.CreatedAt.Unix(),
	}

	_, err := s.client.Collection(collectionName).Documents().Upsert(ctx, document)
	if err != nil {
		return fmt.Errorf("upserting document: %w", err)
	}
	return nil
}

func (s *SearchIndex) DeleteProgram(ctx context.Context, programID string) error {
	_, err := s.client.Collection(collectionName).Document(programID).Delete(ctx)
	if err != nil {
		return fmt.Errorf("deleting document: %w", err)
	}
	return nil
}

func (s *SearchIndex) SearchPrograms(ctx context.Context, params port.SearchParams) (*port.SearchResult, error) {
	page := params.Page
	perPage := params.PerPage

	searchParams := &api.SearchCollectionParams{
		Q:       params.Query,
		QueryBy: "title,description",
		Page:    &page,
		PerPage: &perPage,
	}

	filterBy := s.buildFilter(params.ProgramType, params.Language, params.Tags)
	if filterBy != "" {
		searchParams.FilterBy = &filterBy
	}

	sortBy := s.buildSort(params.Sort)
	if sortBy != "" {
		searchParams.SortBy = &sortBy
	}

	facetBy := "type,language,tags"
	searchParams.FacetBy = &facetBy

	result, err := s.client.Collection(collectionName).Documents().Search(ctx, searchParams)
	if err != nil {
		return nil, fmt.Errorf("searching documents: %w", err)
	}

	totalFound := 0
	if result.Found != nil {
		totalFound = int(*result.Found)
	}

	totalPages := int(math.Ceil(float64(totalFound) / float64(params.PerPage)))

	return &port.SearchResult{
		Results: s.extractPrograms(result),
		Facets:  s.extractFacets(result),
		Pagination: port.Pagination{
			Total:      totalFound,
			Page:       params.Page,
			PerPage:    params.PerPage,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *SearchIndex) GetProgram(ctx context.Context, programID string) (*domain.Program, error) {
	doc, err := s.client.Collection(collectionName).Document(programID).Retrieve(ctx)
	if err != nil {
		return nil, fmt.Errorf("retrieving document: %w", err)
	}

	createdAt := time.Unix(int64(doc["created_at"].(float64)), 0)
	publishedAt := time.Unix(int64(doc["published_at"].(float64)), 0)

	program := &domain.Program{
		ID:          doc["id"].(string),
		Slug:        doc["slug"].(string),
		Title:       doc["title"].(string),
		Description: doc["description"].(string),
		Type:        domain.ProgramType(doc["type"].(string)),
		Language:    domain.ProgramLanguage(doc["language"].(string)),
		DurationMs:  int(doc["duration_ms"].(float64)),
		Tags:        extractTags(doc["tags"]),
		CreatedAt:   createdAt,
		PublishedAt: &publishedAt,
	}

	return program, nil
}

func (s *SearchIndex) GetRecentPrograms(ctx context.Context, params port.RecentParams) (*port.RecentResult, error) {
	page := params.Page
	perPage := params.PerPage

	searchParams := &api.SearchCollectionParams{
		Q:       "",
		QueryBy: "title",
		Page:    &page,
		PerPage: &perPage,
	}

	sortBy := "published_at:desc"
	searchParams.SortBy = &sortBy

	filterBy := s.buildFilter(params.ProgramType, params.Language, nil)
	if filterBy != "" {
		searchParams.FilterBy = &filterBy
	}

	result, err := s.client.Collection(collectionName).Documents().Search(ctx, searchParams)
	if err != nil {
		return nil, fmt.Errorf("fetching recent programs: %w", err)
	}

	totalFound := 0
	if result.Found != nil {
		totalFound = int(*result.Found)
	}

	return &port.RecentResult{
		Results: s.extractPrograms(result),
		Pagination: port.Pagination{
			Total:   totalFound,
			Page:    params.Page,
			PerPage: params.PerPage,
		},
	}, nil
}

func (s *SearchIndex) GetFacets(ctx context.Context) (*port.FacetsResult, error) {
	page := 1
	perPage := 1

	searchParams := &api.SearchCollectionParams{
		Q:       "",
		QueryBy: "title",
		Page:    &page,
		PerPage: &perPage,
	}

	facetBy := "type,language,tags"
	searchParams.FacetBy = &facetBy

	result, err := s.client.Collection(collectionName).Documents().Search(ctx, searchParams)
	if err != nil {
		return nil, fmt.Errorf("fetching facets: %w", err)
	}

	facets := s.extractFacets(result)
	return &port.FacetsResult{
		Facets: *facets,
	}, nil
}

func (s *SearchIndex) CreateCollectionIfNotExists(ctx context.Context) error {
	_, err := s.client.Collection(collectionName).Retrieve(ctx)
	if err == nil {
		return nil
	}

	facet := true
	publishedAt := "published_at"
	schema := &api.CollectionSchema{
		Name: collectionName,
		Fields: []api.Field{
			{Name: "slug", Type: "string"},
			{Name: "title", Type: "string"},
			{Name: "description", Type: "string"},
			{Name: "type", Type: "string", Facet: &facet},
			{Name: "language", Type: "string", Facet: &facet},
			{Name: "duration_ms", Type: "int32"},
			{Name: "tags", Type: "string[]", Facet: &facet},
			{Name: "published_at", Type: "int64"},
			{Name: "created_at", Type: "int64"},
		},
		DefaultSortingField: &publishedAt,
	}

	_, err = s.client.Collections().Create(ctx, schema)
	if err != nil {
		return fmt.Errorf("creating collection: %w", err)
	}

	return nil
}

func (s *SearchIndex) buildFilter(programType, language *string, tags []string) string {
	filter := ""

	if programType != nil {
		filter += fmt.Sprintf("type:=%s", *programType)
	}

	if language != nil {
		if filter != "" {
			filter += " && "
		}
		filter += fmt.Sprintf("language:=%s", *language)
	}

	if len(tags) > 0 {
		if filter != "" {
			filter += " && "
		}
		tagFilter := ""
		for i, tag := range tags {
			if i > 0 {
				tagFilter += " || "
			}
			tagFilter += fmt.Sprintf("tags:=%s", tag)
		}
		filter += fmt.Sprintf("(%s)", tagFilter)
	}

	return filter
}

func (s *SearchIndex) buildSort(sort *string) string {
	if sort == nil {
		return ""
	}

	switch *sort {
	case "recent":
		return "published_at:desc"
	case "oldest":
		return "published_at:asc"
	case "relevance":
		return ""
	default:
		return ""
	}
}

func (s *SearchIndex) extractPrograms(result *api.SearchResult) []port.ProgramResult {
	programs := make([]port.ProgramResult, 0)
	if result.Hits != nil {
		for _, hit := range *result.Hits {
			if hit.Document != nil {
				doc := *hit.Document
				program := port.ProgramResult{
					ID:          doc["id"].(string),
					Slug:        doc["slug"].(string),
					Title:       doc["title"].(string),
					Description: doc["description"].(string),
					Type:        domain.ProgramType(doc["type"].(string)),
					Language:    domain.ProgramLanguage(doc["language"].(string)),
					DurationMs:  int(doc["duration_ms"].(float64)),
					Tags:        extractTags(doc["tags"]),
					PublishedAt: int64(doc["published_at"].(float64)),
					CreatedAt:   int64(doc["created_at"].(float64)),
				}
				programs = append(programs, program)
			}
		}
	}
	return programs
}

func (s *SearchIndex) extractFacets(result *api.SearchResult) *port.Facets {
	facets := &port.Facets{
		Type:     make(map[string]int),
		Language: make(map[string]int),
		Tags:     make(map[string]int),
	}

	if result.FacetCounts != nil && len(*result.FacetCounts) > 0 {
		for _, facet := range *result.FacetCounts {
			if facet.FieldName == nil || facet.Counts == nil {
				continue
			}

			fieldName := *facet.FieldName
			for _, count := range *facet.Counts {
				if count.Count == nil || count.Value == nil || *count.Count == 0 {
					continue
				}

				switch fieldName {
				case "type":
					facets.Type[*count.Value] = int(*count.Count)
				case "language":
					facets.Language[*count.Value] = int(*count.Count)
				case "tags":
					facets.Tags[*count.Value] = int(*count.Count)
				}
			}
		}
	}

	return facets
}

func extractTags(v interface{}) []string {
	if arr, ok := v.([]interface{}); ok {
		tags := make([]string, len(arr))
		for i, item := range arr {
			tags[i] = item.(string)
		}
		return tags
	}
	return []string{}
}
