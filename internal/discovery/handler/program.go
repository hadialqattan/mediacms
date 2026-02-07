package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hadialqattan/mediacms/internal/discovery"
	"github.com/hadialqattan/mediacms/internal/discovery/port"
)

type ProgramHandler struct {
	service *discovery.Service
}

func NewProgramHandler(service *discovery.Service) *ProgramHandler {
	return &ProgramHandler{service: service}
}

func (h *ProgramHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	page := 1
	perPage := 10

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

	categories := r.URL.Query()["category"]
	if len(categories) > 0 {
		params.Categories = categories
	}

	result, err := h.service.SearchPrograms(r.Context(), params)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to search programs: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *ProgramHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	program, err := h.service.GetProgram(r.Context(), id)
	if err != nil {
		http.Error(w, "Program not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(program)
}

func parsePage(s string) (int, error) {
	var page int
	if err := json.Unmarshal([]byte(s), &page); err != nil {
		return 0, err
	}
	if page < 1 {
		return 0, nil
	}
	return page, nil
}

func parsePerPage(s string) (int, error) {
	var perPage int
	if err := json.Unmarshal([]byte(s), &perPage); err != nil {
		return 0, err
	}
	if perPage < 1 || perPage > 100 {
		return 10, nil
	}
	return perPage, nil
}
