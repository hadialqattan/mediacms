package port

import (
	"context"
	"time"
)

type SessionRepo interface {
	CreateSession(ctx context.Context, userID string) (sessionID string, expiresAt time.Time, err error)
	SessionExists(ctx context.Context, sessionID string) (bool, error)
	DeleteSession(ctx context.Context, sessionID string) error
}
