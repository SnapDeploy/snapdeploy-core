package project

import (
	"snapdeploy-core/internal/domain/events"
)

// Event types
const (
	EventTypeProjectCreated = "project.created"
	EventTypeProjectUpdated = "project.updated"
	EventTypeProjectDeleted = "project.deleted"
)

// ProjectCreated is raised when a new project is created
type ProjectCreated struct {
	events.BaseEvent
	ProjectID     string
	UserID        string
	RepositoryURL string
	Language      string
}

func NewProjectCreated(projectID, userID, repositoryURL, language string) *ProjectCreated {
	return &ProjectCreated{
		BaseEvent:     events.NewBaseEvent(EventTypeProjectCreated, projectID),
		ProjectID:     projectID,
		UserID:        userID,
		RepositoryURL: repositoryURL,
		Language:      language,
	}
}

// ProjectUpdated is raised when a project is updated
type ProjectUpdated struct {
	events.BaseEvent
	ProjectID     string
	UserID        string
	RepositoryURL string
	Language      string
}

func NewProjectUpdated(projectID, userID, repositoryURL, language string) *ProjectUpdated {
	return &ProjectUpdated{
		BaseEvent:     events.NewBaseEvent(EventTypeProjectUpdated, projectID),
		ProjectID:     projectID,
		UserID:        userID,
		RepositoryURL: repositoryURL,
		Language:      language,
	}
}

// ProjectDeleted is raised when a project is deleted
type ProjectDeleted struct {
	events.BaseEvent
	ProjectID string
	UserID    string
}

func NewProjectDeleted(projectID, userID string) *ProjectDeleted {
	return &ProjectDeleted{
		BaseEvent: events.NewBaseEvent(EventTypeProjectDeleted, projectID),
		ProjectID: projectID,
		UserID:    userID,
	}
}
