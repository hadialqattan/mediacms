package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/hadialqattan/mediacms/config"
)

type sessionData struct {
	UserID    string `json:"user_id"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at"`
}

type SessionRepo struct {
	client          *redis.Client
	refreshTokenTTL time.Duration
}

func NewSessionRepo(client *redis.Client, cfg config.JWTConfig) *SessionRepo {
	return &SessionRepo{
		client:          client,
		refreshTokenTTL: cfg.RefreshTokenTTL,
	}
}

func (r *SessionRepo) CreateSession(ctx context.Context, userID string) (sessionID string, expiresAt time.Time, err error) {
	sessionID = generateSessionID()
	now := time.Now()
	expiresAt = now.Add(r.refreshTokenTTL)

	data := sessionData{
		UserID:    userID,
		CreatedAt: now.Format(time.RFC3339),
		ExpiresAt: expiresAt.Format(time.RFC3339),
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		return "", time.Time{}, err
	}

	key := "session:" + sessionID
	if err := r.client.Set(ctx, key, bytes, r.refreshTokenTTL).Err(); err != nil {
		return "", time.Time{}, err
	}

	return sessionID, expiresAt, nil
}

func (r *SessionRepo) SessionExists(ctx context.Context, sessionID string) (bool, error) {
	key := "session:" + sessionID
	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}

	return exists > 0, nil
}

func (r *SessionRepo) DeleteSession(ctx context.Context, sessionID string) error {
	key := "session:" + sessionID
	return r.client.Del(ctx, key).Err()
}

func generateSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
