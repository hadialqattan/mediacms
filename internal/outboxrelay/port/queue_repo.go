package port

import (
	"context"

	"github.com/hadialqattan/mediacms/internal/shared/domain"
)

type Queue interface {
	Enqueue(ctx context.Context, eventType domain.OutboxEventType, payload []byte) error
}
