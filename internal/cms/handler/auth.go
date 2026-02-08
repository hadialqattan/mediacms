package handler

import (
	"encoding/json"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"github.com/hadialqattan/mediacms/internal/cms/auth"
	"github.com/hadialqattan/mediacms/internal/cms/repository/sqlc"
	"github.com/hadialqattan/mediacms/internal/cms/service"
	"github.com/hadialqattan/mediacms/internal/shared/domain"
)

type AuthHandler struct {
	svc        *service.Service
	jwtManager *auth.JWTManager
}

func NewAuthHandler(svc *service.Service, jwtManager *auth.JWTManager) *AuthHandler {
	return &AuthHandler{svc: svc, jwtManager: jwtManager}
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// Login authenticates a user and returns JWT tokens
// @Summary      Login
// @Description  Authenticate user with email and password, returns access and refresh tokens
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body handler.LoginRequest true "Login credentials"
// @Success      200 {object} handler.LoginResponse
// @Failure      401 {string} string "Invalid credentials"
// @Router       /api/v1/auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	_, tokens, err := h.svc.Login(r.Context(), req.Email, req.Password, func(hash, password string) error {
		return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	})
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	})
}

// CreateUser creates a new user account (admin only)
// @Summary      Create user
// @Description  Create a new user account with editor role (admin only)
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body handler.LoginRequest true "User credentials"
// @Success      200 {object} handler.LoginResponse
// @Failure      400 {string} string "Invalid request"
// @Failure      409 {string} string "User with this email already exists"
// @Failure      500 {string} string "Failed to create user"
// @Router       /api/v1/users [post]
func (h *AuthHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if _, err := h.svc.GetUserByEmail(r.Context(), req.Email); err == nil {
		http.Error(w, "User with this email already exists", http.StatusConflict)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	user, tokens, err := h.svc.Register(r.Context(), sqlc.CreateUserParams{
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         string(domain.UserRoleEditor),
	})
	if err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	_ = user

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	})
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type RefreshResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

// Refresh exchanges a refresh token for a new access token
// @Summary      Refresh token
// @Description  Exchange a valid refresh token for a new access token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body handler.RefreshRequest true "Refresh token"
// @Success      200 {object} handler.RefreshResponse
// @Failure      401 {string} string "Invalid refresh token"
// @Router       /api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	claims, err := h.jwtManager.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
		return
	}

	accessToken, err := h.svc.RefreshAccessToken(r.Context(), claims.SessionID, claims.UserID)
	if err != nil {
		http.Error(w, "Failed to refresh token", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(RefreshResponse{
		AccessToken: accessToken,
		ExpiresIn:   int64(h.jwtManager.GetAccessTokenTTL().Seconds()),
	})
}

// Logout invalidates the refresh token
// @Summary      Logout
// @Description  Invalidate the refresh token to logout the user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body handler.RefreshRequest true "Refresh token to invalidate"
// @Success      204 "No content"
// @Failure      401 {string} string "Invalid refresh token"
// @Failure      500 {string} string "Failed to logout"
// @Router       /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	claims, err := h.jwtManager.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
		return
	}

	if err := h.svc.Logout(r.Context(), claims.SessionID); err != nil {
		http.Error(w, "Failed to logout", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
