package repository

import (
	"context"

	"github.com/hibiken/asynq"

	"thmanyah.com/content-platform/internal/outboxrelay/port"
	"thmanyah.com/content-platform/internal/shared/domain"
)

type queueRepo struct {
	client *asynq.Client
}

func NewQueue(redisAddr string) port.Queue {
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})
	return &queueRepo{client: client}
}

func (a *queueRepo) Enqueue(ctx context.Context, eventType domain.OutboxEventType, payload []byte) error {
	task := asynq.NewTask(string(eventType), payload)
	_, err := a.client.Enqueue(task)
	return err
}
