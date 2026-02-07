package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hadialqattan/mediacms/config"
)

func TestGenerateAccessToken(t *testing.T) {
	cfg := config.JWTConfig{
		Secret:          "test-secret-key",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	}
	manager := NewJWTManager(cfg)

	userID := "user-123"
	email := "test@example.com"
	role := "admin"

	token, err := manager.GenerateAccessToken(userID, email, role)

	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims := &AccessClaims{}
	parsedToken, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.Secret), nil
	})

	require.NoError(t, err)
	assert.True(t, parsedToken.Valid)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, role, claims.Role)
	assert.NotNil(t, claims.ExpiresAt)
	assert.NotNil(t, claims.IssuedAt)
}

func TestGenerateRefreshToken(t *testing.T) {
	cfg := config.JWTConfig{
		Secret:          "test-secret-key",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	}
	manager := NewJWTManager(cfg)

	sessionID := "session-456"
	userID := "user-123"

	token, err := manager.GenerateRefreshToken(sessionID, userID)

	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims := &RefreshClaims{}
	parsedToken, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.Secret), nil
	})

	require.NoError(t, err)
	assert.True(t, parsedToken.Valid)
	assert.Equal(t, sessionID, claims.SessionID)
	assert.Equal(t, userID, claims.UserID)
	assert.NotNil(t, claims.ExpiresAt)
	assert.NotNil(t, claims.IssuedAt)
}

func TestValidateAccessToken_Valid(t *testing.T) {
	cfg := config.JWTConfig{
		Secret:          "test-secret-key",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	}
	manager := NewJWTManager(cfg)

	userID := "user-123"
	email := "test@example.com"
	role := "editor"

	token, err := manager.GenerateAccessToken(userID, email, role)
	require.NoError(t, err)

	claims, err := manager.ValidateAccessToken(token)

	require.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, email, claims.Email)
	assert.Equal(t, role, claims.Role)
}

func TestValidateAccessToken_Invalid(t *testing.T) {
	cfg := config.JWTConfig{
		Secret:          "test-secret-key",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	}
	manager := NewJWTManager(cfg)

	invalidTokens := []struct {
		name  string
		token string
	}{
		{
			name:  "completely invalid token",
			token: "invalid.token.string",
		},
		{
			name:  "empty token",
			token: "",
		},
		{
			name:  "token with wrong signature",
			token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoidXNlci0xMjMiLCJlbWFpbCI6InRlc3RAZXhhbXBsZS5jb20iLCJyb2xlIjoiYWRtaW4ifQ.invalid-signature",
		},
	}

	for _, tt := range invalidTokens {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := manager.ValidateAccessToken(tt.token)

			assert.Error(t, err)
			assert.Nil(t, claims)
		})
	}
}

func TestValidateAccessToken_Expired(t *testing.T) {
	cfg := config.JWTConfig{
		Secret:          "test-secret-key",
		AccessTokenTTL:  -1 * time.Hour,
		RefreshTokenTTL: 24 * time.Hour,
	}
	manager := NewJWTManager(cfg)

	userID := "user-123"
	email := "test@example.com"
	role := "admin"

	token, err := manager.GenerateAccessToken(userID, email, role)
	require.NoError(t, err)

	claims, err := manager.ValidateAccessToken(token)
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestValidateRefreshToken_Valid(t *testing.T) {
	cfg := config.JWTConfig{
		Secret:          "test-secret-key",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	}
	manager := NewJWTManager(cfg)

	sessionID := "session-456"
	userID := "user-123"

	token, err := manager.GenerateRefreshToken(sessionID, userID)
	require.NoError(t, err)

	claims, err := manager.ValidateRefreshToken(token)
	require.NoError(t, err)
	assert.Equal(t, sessionID, claims.SessionID)
	assert.Equal(t, userID, claims.UserID)
}

func TestValidateRefreshToken_Invalid(t *testing.T) {
	cfg := config.JWTConfig{
		Secret:          "test-secret-key",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	}
	manager := NewJWTManager(cfg)

	invalidTokens := []struct {
		name  string
		token string
	}{
		{
			name:  "completely invalid token",
			token: "invalid.token.string",
		},
		{
			name:  "empty token",
			token: "",
		},
		{
			name:  "malformed token",
			token: "not-a-jwt-token",
		},
	}

	for _, tt := range invalidTokens {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := manager.ValidateRefreshToken(tt.token)

			assert.Error(t, err)
			assert.Nil(t, claims)
		})
	}
}

func TestValidateRefreshToken_Expired(t *testing.T) {
	cfg := config.JWTConfig{
		Secret:          "test-secret-key",
		AccessTokenTTL:  1 * time.Hour,
		RefreshTokenTTL: -24 * time.Hour,
	}
	manager := NewJWTManager(cfg)

	sessionID := "session-456"
	userID := "user-123"

	token, err := manager.GenerateRefreshToken(sessionID, userID)
	require.NoError(t, err)

	claims, err := manager.ValidateRefreshToken(token)
	require.Error(t, err)
	assert.Nil(t, claims)
}
