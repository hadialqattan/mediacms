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
