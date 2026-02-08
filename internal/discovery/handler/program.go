package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/hadialqattan/mediacms/internal/discovery"
	"github.com/hadialqattan/mediacms/internal/discovery/port"
)

type ProgramHandler struct {
	service *discovery.Service
}

func NewProgramHandler(service *discovery.Service) *ProgramHandler {
	return &ProgramHandler{
		service: service,
	}
}

// ProgramResult represents a program in search results
type ProgramResult struct {
	ID          string   `json:"id"`
	Slug        string   `json:"slug"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
	Language    string   `json:"language"`
	DurationMs  int      `json:"duration_ms"`
	Tags        []string `json:"tags"`
	PublishedAt int64    `json:"published_at"`
	CreatedAt   int64    `json:"created_at"`
}

// SearchFacets represents search facets
type SearchFacets struct {
	Type     map[string]int `json:"type,omitempty"`
	Language map[string]int `json:"language,omitempty"`
	Tags     map[string]int `json:"tags,omitempty"`
}

// SearchPagination represents pagination info
type SearchPagination struct {
	Total      int `json:"total"`
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
}

// SearchProgramsResponse represents the search response
type SearchProgramsResponse struct {
	Results    []ProgramResult  `json:"results"`
	Facets     *SearchFacets    `json:"facets,omitempty"`
	Pagination SearchPagination `json:"pagination"`
}

// RecentProgramsResponse represents the recent programs response
type RecentProgramsResponse struct {
	Results    []ProgramResult  `json:"results"`
	Pagination SearchPagination `json:"pagination"`
}

// FacetsResponse represents the facets response
type FacetsResponse struct {
	Facets SearchFacets `json:"facets"`
}

// SearchPrograms searches programs with filters and pagination
// @Summary      Search programs
// @Description  Search programs with filters and pagination
// @Tags         programs
// @Accept       json
// @Produce      json
// @Param        q query string false "Search query"
// @Param        page query int false "Page number" minimum(1) default(1)
// @Param        per_page query int false "Items per page" minimum(1) maximum(100) default(20)
// @Param        type query string false "Filter by type" Enums(podcast,documentary)
// @Param        language query string false "Filter by language" Enums(ar,en)
// @Param        sort query string false "Sort order" Enums(published_at_desc,published_at_asc,relevance)
// @Param        tags query string false "Filter by tags (comma-separated)"
// @Success      200 {object} handler.SearchProgramsResponse
// @Router       /api/v1/programs [get]
func (h *ProgramHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	page := 1
	perPage := 20

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := parsePage(pageStr); err == nil {
			page = p
		}
	}

	if perPageStr := r.URL.Query().Get("per_page"); perPageStr != "" {
		if pp, err := parsePerPage(perPageStr); err == nil {
			perPage = pp
		}
	}

	params := port.SearchParams{
		Query:   query,
		Page:    page,
		PerPage: perPage,
	}

	if programType := r.URL.Query().Get("type"); programType != "" {
		params.ProgramType = &programType
	}

	if language := r.URL.Query().Get("language"); language != "" {
		params.Language = &language
	}

	if sort := r.URL.Query().Get("sort"); sort != "" {
		params.Sort = &sort
	}

	if tagsStr := r.URL.Query().Get("tags"); tagsStr != "" {
		params.Tags = splitTags(tagsStr)
	}

	result, err := h.service.SearchPrograms(r.Context(), params)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to search programs: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=10")
	json.NewEncoder(w).Encode(result)
}

// GetProgram retrieves a published program by ID
// @Summary      Get program
// @Description  Get a specific published program by its ID
// @Tags         programs
// @Accept       json
// @Produce      json
// @Param        id path string true "Program ID"
// @Success      200 {object} handler.ProgramResult
// @Failure      404 {string} string "Program not found"
// @Router       /api/v1/programs/{id} [get]
func (h *ProgramHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	program, err := h.service.GetProgram(r.Context(), id)
	if err != nil {
		http.Error(w, "Program not found", http.StatusNotFound)
		return
	}

	createdAt := program.CreatedAt.Unix()
	publishedAt := program.PublishedAt.Unix()
	response := map[string]interface{}{
		"id":           program.ID,
		"slug":         program.Slug,
		"title":        program.Title,
		"description":  program.Description,
		"type":         program.Type,
		"language":     program.Language,
		"duration_ms":  program.DurationMs,
		"tags":         program.Tags,
		"published_at": publishedAt,
		"created_at":   createdAt,
	}

	etag := fmt.Sprintf("%s-%d", program.ID, createdAt)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300, stale-while-revalidate=600")
	w.Header().Set("ETag", etag)
	json.NewEncoder(w).Encode(response)
}

// GetRecentPrograms retrieves recently published programs
// @Summary      Get recent programs
// @Description  Get recently published programs with optional filters
// @Tags         programs
// @Accept       json
// @Produce      json
// @Param        page query int false "Page number" minimum(1) default(1)
// @Param        per_page query int false "Items per page" minimum(1) maximum(100) default(20)
// @Param        type query string false "Filter by type" Enums(podcast,documentary)
// @Param        language query string false "Filter by language" Enums(ar,en)
// @Success      200 {object} handler.RecentProgramsResponse
// @Router       /api/v1/programs/recent [get]
func (h *ProgramHandler) GetRecent(w http.ResponseWriter, r *http.Request) {
	page := 1
	perPage := 20

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := parsePage(pageStr); err == nil {
			page = p
		}
	}

	if perPageStr := r.URL.Query().Get("per_page"); perPageStr != "" {
		if pp, err := parsePerPage(perPageStr); err == nil {
			perPage = pp
		}
	}

	params := port.RecentParams{
		Page:    page,
		PerPage: perPage,
	}

	if programType := r.URL.Query().Get("type"); programType != "" {
		params.ProgramType = &programType
	}

	if language := r.URL.Query().Get("language"); language != "" {
		params.Language = &language
	}

	result, err := h.service.GetRecentPrograms(r.Context(), params)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get recent programs: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=60, stale-while-revalidate=300")
	json.NewEncoder(w).Encode(result)
}

// GetFacets retrieves available facets for filtering programs
// @Summary      Get facets
// @Description  Get available facets (types, languages, tags) for filtering programs
// @Tags         programs
// @Accept       json
// @Produce      json
// @Success      200 {object} handler.FacetsResponse
// @Router       /api/v1/programs/facets [get]
func (h *ProgramHandler) GetFacets(w http.ResponseWriter, r *http.Request) {
	facets, err := h.service.GetFacets(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get facets: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300, stale-while-revalidate=1800")
	json.NewEncoder(w).Encode(facets)
}

func parsePage(s string) (int, error) {
	page, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	if page < 1 {
		return 0, nil
	}
	return page, nil
}

func parsePerPage(s string) (int, error) {
	perPage, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	if perPage < 1 || perPage > 100 {
		return 20, nil
	}
	return perPage, nil
}

func splitTags(s string) []string {
	tags := strings.Split(s, ",")
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
