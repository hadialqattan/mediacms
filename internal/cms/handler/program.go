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

type CreateProgramRequest struct {
	Slug        string                 `json:"slug" example:"my-podcast"`
	Title       string                 `json:"title" example:"My Podcast"`
	Description string                 `json:"description" example:"A great podcast about technology"`
	Type        domain.ProgramType     `json:"type" enums:"podcast,documentary" example:"podcast"`
	Language    domain.ProgramLanguage `json:"language" enums:"ar,en" example:"ar"`
	DurationMs  int                    `json:"duration_ms" example:"3600000"`
	Tags        []string               `json:"tags" example:"news,sports"`
}

// CreateProgram creates a new program
// @Summary      Create program
// @Description  Create a new podcast or documentary program
// @Tags         programs
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body handler.CreateProgramRequest true "Program data"
// @Success      201 {object} domain.Program
// @Failure      400 {string} string "Invalid type or language"
// @Failure      409 {string} string "Program with this slug already exists"
// @Failure      500 {string} string "Failed to create program"
// @Router       /api/v1/programs [post]
func (h *ProgramHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateProgramRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if !isValidSlug(req.Slug) {
		http.Error(w, "slug must contain only lowercase letters, numbers, and hyphens, and cannot start or end with a hyphen", http.StatusBadRequest)
		return
	}

	if _, err := h.svc.GetProgramBySlug(r.Context(), req.Slug); err == nil {
		http.Error(w, "Program with this slug already exists", http.StatusConflict)
		return
	}

	if !domain.IsValidProgramType(string(req.Type)) {
		http.Error(w, "Invalid type: must be 'podcast' or 'documentary'", http.StatusBadRequest)
		return
	}

	if !domain.IsValidProgramLanguage(string(req.Language)) {
		http.Error(w, "Invalid language: must be 'ar' or 'en'", http.StatusBadRequest)
		return
	}

	createParams := sqlc.CreateProgramParams{
		Slug:        req.Slug,
		Title:       req.Title,
		Description: pgtype.Text{String: req.Description, Valid: true},
		Type:        string(req.Type),
		Language:    string(req.Language),
		DurationMs:  int32(req.DurationMs),
		Tags:        req.Tags,
		CreatedBy:   pgtype.UUID{Bytes: uuid.MustParse(middleware.GetUserID(r)), Valid: true},
	}

	program, err := h.svc.CreateProgram(r.Context(), createParams)

	if err != nil {
		http.Error(w, "Failed to create program", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(program)
}

// GetProgram retrieves a program by ID
// @Summary      Get program
// @Description  Get a specific program by its ID
// @Tags         programs
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Program ID"
// @Success      200 {object} domain.Program
// @Failure      404 {string} string "Program not found"
// @Router       /api/v1/programs/{id} [get]
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

// ListPrograms retrieves all programs with pagination
// @Summary      List programs
// @Description  Get a paginated list of all programs
// @Tags         programs
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Success      200 {object} handler.ListProgramsResponse
// @Failure      500 {string} string "Failed to list programs"
// @Router       /api/v1/programs [get]
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
	Title       *string   `json:"title,omitempty"`
	Description *string   `json:"description,omitempty"`
	Type        *string   `json:"type,omitempty"`
	Language    *string   `json:"language,omitempty"`
	DurationMs  *int      `json:"duration_ms,omitempty"`
	Tags        *[]string `json:"tags,omitempty"`
}

// UpdateProgram updates an existing program
// @Summary      Update program
// @Description  Update an existing program by ID
// @Tags         programs
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Program ID"
// @Param        request body handler.UpdateProgramRequest true "Program update data"
// @Success      200 {object} domain.Program
// @Failure      400 {string} string "Invalid type or language"
// @Failure      404 {string} string "Program not found"
// @Failure      500 {string} string "Failed to update program"
// @Router       /api/v1/programs/{id} [put]
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

// PublishProgram publishes a program
// @Summary      Publish program
// @Description  Publish a program to make it available in the discovery API
// @Tags         programs
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Program ID"
// @Success      200 {object} domain.Program
// @Failure      500 {string} string "Failed to publish program"
// @Router       /api/v1/programs/{id}/publish [post]
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

// DeleteProgram deletes a program
// @Summary      Delete program
// @Description  Delete a program by ID (soft delete)
// @Tags         programs
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Program ID"
// @Success      204 "No content"
// @Failure      500 {string} string "Failed to delete program"
// @Router       /api/v1/programs/{id} [delete]
func (h *ProgramHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.svc.DeleteProgram(r.Context(), id, middleware.GetUserID(r)); err != nil {
		http.Error(w, "Failed to delete program", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type BulkCreateRequest struct {
	Programs []CreateProgramRequest `json:"programs"`
}

type BulkCreateResponse struct {
	Created []*domain.Program                  `json:"created"`
	Failed  []service.BulkCreateProgramFailure `json:"failed"`
}

// BulkCreatePrograms creates multiple programs
// @Summary      Bulk create programs
// @Description  Create multiple programs in a single request
// @Tags         programs
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body handler.BulkCreateRequest true "List of programs to create"
// @Success      207 {object} handler.BulkCreateResponse
// @Failure      400 {string} string "Invalid request"
// @Router       /api/v1/programs/bulk [post]
func (h *ProgramHandler) BulkCreate(w http.ResponseWriter, r *http.Request) {
	var req BulkCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	createdByID := uuid.MustParse(middleware.GetUserID(r))

	programsParams := make([]sqlc.CreateProgramParams, len(req.Programs))
	for i, p := range req.Programs {
		programsParams[i] = sqlc.CreateProgramParams{
			Slug:        p.Slug,
			Title:       p.Title,
			Description: pgtype.Text{String: p.Description, Valid: true},
			Type:        string(p.Type),
			Language:    string(p.Language),
			DurationMs:  int32(p.DurationMs),
			Tags:        p.Tags,
			CreatedBy:   pgtype.UUID{Bytes: createdByID, Valid: true},
		}
	}

	programs, failures := h.svc.BulkCreatePrograms(r.Context(), programsParams, createdByID.String())

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

// BulkDeletePrograms deletes multiple programs
// @Summary      Bulk delete programs
// @Description  Delete multiple programs by their IDs
// @Tags         programs
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body handler.BulkDeleteRequest true "List of program IDs to delete"
// @Success      207 {object} handler.BulkDeleteResponse
// @Failure      400 {string} string "Invalid request"
// @Router       /api/v1/programs/bulk [delete]
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

// Note: This is not supposed to be here (part of router), but for the sake of time, I'll leave it here ;-).
func isValidSlug(slug string) bool {
	if slug == "" {
		return false
	}

	if slug[0] == '-' || slug[len(slug)-1] == '-' {
		return false
	}

	prevWasHyphen := false
	for i, r := range slug {
		if r == '-' {
			if prevWasHyphen {
				return false
			}
			prevWasHyphen = true
		} else if r >= 'a' && r <= 'z' {
			prevWasHyphen = false
		} else if r >= '0' && r <= '9' && i > 0 {
			prevWasHyphen = false
		} else {
			return false
		}
	}

	if prevWasHyphen {
		return false
	}

	return true
}
