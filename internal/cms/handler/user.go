package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/hadialqattan/mediacms/internal/cms/repository/sqlc"
	"github.com/hadialqattan/mediacms/internal/cms/service"
	"github.com/hadialqattan/mediacms/internal/shared/domain"
)

type UserHandler struct {
	svc *service.Service
}

func NewUserHandler(svc *service.Service) *UserHandler {
	return &UserHandler{svc: svc}
}

type CreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type CreateUserResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		// TODO: Should be real validation but for simplicity I'll keep it like this.
		http.Error(w, "email and password are required", http.StatusBadRequest)
		return
	}

	userRole := domain.UserRole(req.Role)
	if !domain.IsValidUserRole(req.Role) {
		http.Error(w, "invalid role (must be 'admin' or 'editor')", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	user, err := h.svc.CreateUser(r.Context(), sqlc.CreateUserParams{
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         string(userRole),
	})
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			http.Error(w, "User with this email already exists", http.StatusConflict)
			return
		}
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(CreateUserResponse{
		ID:    user.ID,
		Email: user.Email,
		Role:  string(user.Role),
	})
}
