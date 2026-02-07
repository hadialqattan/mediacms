package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/hadialqattan/mediacms/config"
)

type JWTManager struct {
	secret          string
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func NewJWTManager(cfg config.JWTConfig) *JWTManager {
	return &JWTManager{
		secret:          cfg.Secret,
		accessTokenTTL:  cfg.AccessTokenTTL,
		refreshTokenTTL: cfg.RefreshTokenTTL,
	}
}

type AccessClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type RefreshClaims struct {
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`
	jwt.RegisteredClaims
}

func (m *JWTManager) GenerateAccessToken(userID, email, role string) (string, error) {
	now := time.Now()
	claims := AccessClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.secret))
}

func (m *JWTManager) GenerateRefreshToken(sessionID, userID string) (string, error) {
	now := time.Now()
	claims := RefreshClaims{
		SessionID: sessionID,
		UserID:    userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.secret))
}

func (m *JWTManager) ValidateAccessToken(tokenString string) (*AccessClaims, error) {
	claims := &AccessClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(m.secret), nil
	})

	if err != nil || !token.Valid {
		return nil, err
	}

	return claims, nil
}

func (m *JWTManager) ValidateRefreshToken(tokenString string) (*RefreshClaims, error) {
	claims := &RefreshClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(m.secret), nil
	})

	if err != nil || !token.Valid {
		return nil, err
	}

	return claims, nil
}

func (m *JWTManager) GetAccessTokenTTL() time.Duration {
	return m.accessTokenTTL
}
