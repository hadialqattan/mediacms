package port

import (
	"context"
)

type MessageQueue interface {
	Enqueue(ctx context.Context, eventType string, payload []byte) error
}
