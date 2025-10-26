package events

import (
	"time"

	"github.com/google/uuid"
)

// DomainEvent represents a domain event
type DomainEvent interface {
	EventID() string
	EventType() string
	OccurredAt() time.Time
	AggregateID() string
}

// BaseEvent provides common event properties
type BaseEvent struct {
	eventID     string
	eventType   string
	occurredAt  time.Time
	aggregateID string
}

// NewBaseEvent creates a new base event
func NewBaseEvent(eventType, aggregateID string) BaseEvent {
	return BaseEvent{
		eventID:     uuid.New().String(),
		eventType:   eventType,
		occurredAt:  time.Now(),
		aggregateID: aggregateID,
	}
}

func (e BaseEvent) EventID() string {
	return e.eventID
}

func (e BaseEvent) EventType() string {
	return e.eventType
}

func (e BaseEvent) OccurredAt() time.Time {
	return e.occurredAt
}

func (e BaseEvent) AggregateID() string {
	return e.aggregateID
}
