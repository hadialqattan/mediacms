package repository

import (
	"context"

	"github.com/hibiken/asynq"

	"thmanyah.com/content-platform/internal/outboxrelay/port"
)

type asynqClient struct {
	client *asynq.Client
}

func NewAsynqClient(redisAddr string) port.MessageQueue {
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})
	return &asynqClient{client: client}
}

func (a *asynqClient) Enqueue(ctx context.Context, eventType string, payload []byte) error {
	task := asynq.NewTask(eventType, payload)
	_, err := a.client.Enqueue(task)
	return err
}
