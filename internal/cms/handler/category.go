package handler

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hadialqattan/mediacms/internal/cms/repository/sqlc"
	"github.com/hadialqattan/mediacms/internal/cms/service"
)

type CategoryHandler struct {
	svc *service.Service
}

func NewCategoryHandler(svc *service.Service) *CategoryHandler {
	return &CategoryHandler{svc: svc}
}

type CreateCategoryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *CategoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	category, err := h.svc.CreateCategory(r.Context(), sqlc.CreateCategoryParams{
		Name:        req.Name,
		Description: pgtype.Text{String: req.Description, Valid: req.Description != ""},
	})
	if err != nil {
		http.Error(w, "Failed to create category", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(category)
}

func (h *CategoryHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	category, err := h.svc.GetCategory(r.Context(), id)
	if err != nil {
		http.Error(w, "Category not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(category)
}

func (h *CategoryHandler) List(w http.ResponseWriter, r *http.Request) {
	categories, err := h.svc.ListCategories(r.Context())
	if err != nil {
		http.Error(w, "Failed to list categories", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categories)
}
