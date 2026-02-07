package handler

import (
	"encoding/json"
	"net/http"

	"github.com/hadialqattan/mediacms/internal/cms/middleware"
	"github.com/hadialqattan/mediacms/internal/cms/service"
	"github.com/hadialqattan/mediacms/internal/shared/domain"
)

type ImportHandler struct {
	svc *service.Service
}

func NewImportHandler(svc *service.Service) *ImportHandler {
	return &ImportHandler{svc: svc}
}

type ImportRequest struct {
	SourceType string                 `json:"source_type"`
	Metadata   map[string]interface{} `json:"metadata"`
}

func (h *ImportHandler) Import(w http.ResponseWriter, r *http.Request) {
	var req ImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	metadata := req.Metadata
	metadata["created_by"] = middleware.GetUserID(r)

	program, err := h.svc.ImportProgram(r.Context(), domain.SourceType(req.SourceType), metadata)
	if err != nil {
		http.Error(w, "Failed to import program", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(program)
}
