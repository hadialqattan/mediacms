package domain

import "time"

type OutboxEventType string

const (
	OutboxEventTypeProgramUpsert OutboxEventType = "program.upsert"
	OutboxEventTypeProgramDelete OutboxEventType = "program.delete"
)

func IsValidOutboxEventType(typ string) bool {
	switch typ {
	case string(OutboxEventTypeProgramUpsert), string(OutboxEventTypeProgramDelete):
		return true
	default:
		return false
	}
}

type OutboxEvent struct {
	ID        string
	Type      OutboxEventType
	Payload   map[string]interface{}
	Enqueued  bool
	CreatedAt time.Time
	ProgramID *string
}
