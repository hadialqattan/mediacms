package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"thmanyah.com/content-platform/internal/cms/middleware"
	"thmanyah.com/content-platform/internal/cms/repository/sqlc"
	"thmanyah.com/content-platform/internal/cms/service"
	"thmanyah.com/content-platform/internal/shared/domain"
)

type ProgramHandler struct {
	svc *service.Service
}

func NewProgramHandler(svc *service.Service) *ProgramHandler {
	return &ProgramHandler{svc: svc}
}

type CreateProgramRequest struct {
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Language    string `json:"language"`
	DurationMs  int    `json:"duration_ms"`
}

func (h *ProgramHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateProgramRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if !domain.IsValidProgramType(req.Type) {
		http.Error(w, "Invalid type: must be 'podcast' or 'documentary'", http.StatusBadRequest)
		return
	}

	if !domain.IsValidProgramLanguage(req.Language) {
		http.Error(w, "Invalid language: must be 'ar' or 'en'", http.StatusBadRequest)
		return
	}

	createdByID := uuid.MustParse(middleware.GetUserID(r))
	program, err := h.svc.CreateProgram(r.Context(), sqlc.CreateProgramParams{
		Slug:        req.Slug,
		Title:       req.Title,
		Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
		Type:        req.Type,
		Language:    req.Language,
		DurationMs:  int32(req.DurationMs),
		SourceID:    pgtype.UUID{},
		CreatedBy:   pgtype.UUID{Bytes: createdByID, Valid: true},
	})
	if err != nil {
		http.Error(w, "Failed to create program", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(program)
}

func (h *ProgramHandler) Get(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")
	program, err := h.svc.GetProgramBySlug(r.Context(), slug)
	if err != nil {
		http.Error(w, "Program not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(program)
}

func (h *ProgramHandler) List(w http.ResponseWriter, r *http.Request) {
	programs, err := h.svc.ListPrograms(r.Context())
	if err != nil {
		http.Error(w, "Failed to list programs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(programs)
}

type UpdateProgramRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Type        *string `json:"type"`
	Language    *string `json:"language"`
	DurationMs  *int    `json:"duration_ms"`
}

func (h *ProgramHandler) Update(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")

	var req UpdateProgramRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Type != nil && !domain.IsValidProgramType(*req.Type) {
		http.Error(w, "Invalid type: must be 'podcast' or 'documentary'", http.StatusBadRequest)
		return
	}

	if req.Language != nil && !domain.IsValidProgramLanguage(*req.Language) {
		http.Error(w, "Invalid language: must be 'ar' or 'en'", http.StatusBadRequest)
		return
	}

	program, err := h.svc.GetProgramBySlug(r.Context(), slug)
	if err != nil {
		http.Error(w, "Program not found", http.StatusNotFound)
		return
	}

	title := program.Title
	description := pgtype.Text{String: program.Description, Valid: program.Description != ""}
	progType := string(program.Type)
	language := string(program.Language)
	durationMs := int32(program.DurationMs)

	if req.Title != nil {
		title = *req.Title
	}
	if req.Description != nil {
		description = pgtype.Text{String: *req.Description, Valid: true}
	}
	if req.Type != nil {
		progType = *req.Type
	}
	if req.Language != nil {
		language = *req.Language
	}
	if req.DurationMs != nil {
		durationMs = int32(*req.DurationMs)
	}

	updatedByID := uuid.MustParse(middleware.GetUserID(r))
	updateParams := sqlc.UpdateProgramParams{
		ID:          pgtype.UUID{Bytes: uuid.MustParse(program.ID), Valid: true},
		Title:       title,
		Description: description,
		Type:        progType,
		Language:    language,
		DurationMs:  durationMs,
		UpdatedBy:   pgtype.UUID{Bytes: updatedByID, Valid: true},
	}

	updatedProgram, err := h.svc.UpdateProgram(r.Context(), program.ID, updateParams)
	if err != nil {
		http.Error(w, "Failed to update program", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedProgram)
}

func (h *ProgramHandler) Publish(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")

	program, err := h.svc.PublishProgramBySlug(r.Context(), slug, middleware.GetUserID(r))
	if err != nil {
		http.Error(w, "Failed to publish program", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(program)
}

func (h *ProgramHandler) Delete(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")

	if err := h.svc.DeleteProgramBySlug(r.Context(), slug, middleware.GetUserID(r)); err != nil {
		http.Error(w, "Failed to delete program", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type AssignCategoriesRequest struct {
	CategoryIDs []string `json:"category_ids"`
}

func (h *ProgramHandler) AssignCategories(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")

	var req AssignCategoriesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if err := h.svc.AssignCategoriesBySlug(r.Context(), slug, req.CategoryIDs); err != nil {
		http.Error(w, "Failed to assign categories", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ProgramHandler) GetCategories(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("id")

	categories, err := h.svc.GetProgramCategoriesBySlug(r.Context(), slug)
	if err != nil {
		http.Error(w, "Failed to get categories", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categories)
}
