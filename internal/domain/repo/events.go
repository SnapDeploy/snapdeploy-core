package repo

import (
	"snapdeploy-core/internal/domain/events"
)

// Event types
const (
	EventTypeRepositorySynced = "repository.synced"
	EventTypeRepositoryAdded  = "repository.added"
)

// RepositoriesSyncedEvent is raised when repositories are synced from GitHub
type RepositoriesSyncedEvent struct {
	events.BaseEvent
	UserID          string
	RepositoryCount int
}

// NewRepositoriesSyncedEvent creates a new RepositoriesSyncedEvent
func NewRepositoriesSyncedEvent(userID string, count int) *RepositoriesSyncedEvent {
	return &RepositoriesSyncedEvent{
		BaseEvent:       events.NewBaseEvent(EventTypeRepositorySynced, userID),
		UserID:          userID,
		RepositoryCount: count,
	}
}

// RepositoryAddedEvent is raised when a new repository is added
type RepositoryAddedEvent struct {
	events.BaseEvent
	RepositoryID string
	UserID       string
	Name         string
}

// NewRepositoryAddedEvent creates a new RepositoryAddedEvent
func NewRepositoryAddedEvent(repoID, userID, name string) *RepositoryAddedEvent {
	return &RepositoryAddedEvent{
		BaseEvent:    events.NewBaseEvent(EventTypeRepositoryAdded, repoID),
		RepositoryID: repoID,
		UserID:       userID,
		Name:         name,
	}
}
