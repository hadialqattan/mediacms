package port

import (
	"context"

	"thmanyah.com/content-platform/internal/shared/domain"
)

type Queue interface {
	Enqueue(ctx context.Context, eventType domain.OutboxEventType, payload []byte) error
}
