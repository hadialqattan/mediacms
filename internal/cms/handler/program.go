package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hadialqattan/mediacms/internal/cms/middleware"
	"github.com/hadialqattan/mediacms/internal/cms/repository/sqlc"
	"github.com/hadialqattan/mediacms/internal/cms/service"
	"github.com/hadialqattan/mediacms/internal/shared/domain"
)

type ProgramHandler struct {
	svc *service.Service
}

func NewProgramHandler(svc *service.Service) *ProgramHandler {
	return &ProgramHandler{svc: svc}
}

func (h *ProgramHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req sqlc.CreateProgramParams
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

	req.CreatedBy = pgtype.UUID{Bytes: uuid.MustParse(middleware.GetUserID(r)), Valid: true}
	program, err := h.svc.CreateProgram(r.Context(), req)

	if err != nil {
		http.Error(w, "Failed to create program", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(program)
}

func (h *ProgramHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	program, err := h.svc.GetProgram(r.Context(), id)
	if err != nil {
		http.Error(w, "Program not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(program)
}

type ListProgramsResponse struct {
	Data []*domain.Program `json:"data"`
	Meta PaginationMeta    `json:"meta"`
}

type PaginationMeta struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

func (h *ProgramHandler) List(w http.ResponseWriter, r *http.Request) {
	page := 1
	limit := 20

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	programs, err := h.svc.ListPrograms(r.Context(), limit, (page-1)*limit)
	if err != nil {
		http.Error(w, "Failed to list programs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ListProgramsResponse{
		Data: programs,
		Meta: PaginationMeta{
			Page:  page,
			Limit: limit,
		},
	})
}

type UpdateProgramRequest struct {
	Title       *string   `json:"title"`
	Description *string   `json:"description"`
	Type        *string   `json:"type"`
	Language    *string   `json:"language"`
	DurationMs  *int      `json:"duration_ms"`
	Tags        *[]string `json:"tags"`
}

func (h *ProgramHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

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

	program, err := h.svc.GetProgram(r.Context(), id)
	if err != nil {
		http.Error(w, "Program not found", http.StatusNotFound)
		return
	}

	title := program.Title
	description := pgtype.Text{String: program.Description, Valid: program.Description != ""}
	progType := string(program.Type)
	language := string(program.Language)
	durationMs := int32(program.DurationMs)
	tags := program.Tags

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
	if req.Tags != nil {
		tags = *req.Tags
	}

	updatedByID := uuid.MustParse(middleware.GetUserID(r))
	updateParams := sqlc.UpdateProgramParams{
		ID:          pgtype.UUID{Bytes: uuid.MustParse(program.ID), Valid: true},
		Title:       title,
		Description: description,
		Type:        progType,
		Language:    language,
		DurationMs:  durationMs,
		Tags:        tags,
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
	id := r.PathValue("id")

	program, err := h.svc.PublishProgram(r.Context(), id, middleware.GetUserID(r))
	if err != nil {
		http.Error(w, "Failed to publish program", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(program)
}

func (h *ProgramHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.svc.DeleteProgram(r.Context(), id, middleware.GetUserID(r)); err != nil {
		http.Error(w, "Failed to delete program", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type BulkCreateRequest struct {
	Programs []sqlc.CreateProgramParams `json:"programs"`
}

type BulkCreateResponse struct {
	Created []*domain.Program                  `json:"created"`
	Failed  []service.BulkCreateProgramFailure `json:"failed"`
}

func (h *ProgramHandler) BulkCreate(w http.ResponseWriter, r *http.Request) {
	var req BulkCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	createdByID := uuid.MustParse(middleware.GetUserID(r))
	programs, failures := h.svc.BulkCreatePrograms(r.Context(), req.Programs, createdByID.String())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMultiStatus)
	json.NewEncoder(w).Encode(BulkCreateResponse{
		Created: programs,
		Failed:  failures,
	})
}

type BulkDeleteRequest struct {
	IDs []string `json:"ids"`
}

type BulkDeleteResponse struct {
	Deleted []string                           `json:"deleted"`
	Failed  []service.BulkDeleteProgramFailure `json:"failed"`
}

func (h *ProgramHandler) BulkDelete(w http.ResponseWriter, r *http.Request) {
	var req BulkDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	deletedIDs, failures := h.svc.BulkDeletePrograms(r.Context(), req.IDs, middleware.GetUserID(r))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMultiStatus)
	json.NewEncoder(w).Encode(BulkDeleteResponse{
		Deleted: deletedIDs,
		Failed:  failures,
	})
}
