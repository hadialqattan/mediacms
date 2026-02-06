package outboxrelay

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"thmanyah.com/content-platform/internal/outboxrelay/port"
)

type Relay struct {
	outboxRepo port.OutboxRepo
	queue      port.MessageQueue
	interval   time.Duration
}

func NewRelay(outboxRepo port.OutboxRepo, queue port.MessageQueue, interval time.Duration) *Relay {
	return &Relay{
		outboxRepo: outboxRepo,
		queue:      queue,
		interval:   interval,
	}
}

func (r *Relay) Start(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.poll(ctx); err != nil {
				log.Printf("Failed to poll outbox: %v", err)
			}
		}
	}
}

func (r *Relay) poll(ctx context.Context) error {
	events, err := r.outboxRepo.GetPending(ctx)
	if err != nil {
		return err
	}

	for _, event := range events {
		payload, err := json.Marshal(event.Payload)
		if err != nil {
			log.Printf("Failed to marshal payload: %v", err)
			continue
		}

		if err := r.queue.Enqueue(ctx, string(event.Type), payload); err != nil {
			log.Printf("Failed to enqueue task: %v", err)
			continue
		}

		log.Printf("Enqueued task %s: %v", event.ID, event.Type)

		if err := r.outboxRepo.MarkEnqueued(ctx, event.ID); err != nil {
			log.Printf("Failed to mark outbox event as enqueued: %v", err)
		}
	}

	return nil
}
