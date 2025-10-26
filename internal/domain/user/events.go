package user

import (
	"snapdeploy-core/internal/domain/events"
)

// Event types
const (
	EventTypeUserCreated = "user.created"
	EventTypeUserUpdated = "user.updated"
	EventTypeUserDeleted = "user.deleted"
)

// UserCreatedEvent is raised when a new user is created
type UserCreatedEvent struct {
	events.BaseEvent
	UserID      string
	Email       string
	Username    string
	ClerkUserID string
}

// NewUserCreatedEvent creates a new UserCreatedEvent
func NewUserCreatedEvent(userID, email, username, clerkUserID string) *UserCreatedEvent {
	return &UserCreatedEvent{
		BaseEvent:   events.NewBaseEvent(EventTypeUserCreated, userID),
		UserID:      userID,
		Email:       email,
		Username:    username,
		ClerkUserID: clerkUserID,
	}
}

// UserUpdatedEvent is raised when a user is updated
type UserUpdatedEvent struct {
	events.BaseEvent
	UserID   string
	Email    string
	Username string
}

// NewUserUpdatedEvent creates a new UserUpdatedEvent
func NewUserUpdatedEvent(userID, email, username string) *UserUpdatedEvent {
	return &UserUpdatedEvent{
		BaseEvent: events.NewBaseEvent(EventTypeUserUpdated, userID),
		UserID:    userID,
		Email:     email,
		Username:  username,
	}
}

// UserDeletedEvent is raised when a user is deleted
type UserDeletedEvent struct {
	events.BaseEvent
	UserID string
}

// NewUserDeletedEvent creates a new UserDeletedEvent
func NewUserDeletedEvent(userID string) *UserDeletedEvent {
	return &UserDeletedEvent{
		BaseEvent: events.NewBaseEvent(EventTypeUserDeleted, userID),
		UserID:    userID,
	}
}
