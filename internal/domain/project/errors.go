package project

import "errors"

var (
	// ErrProjectNotFound is returned when a project is not found
	ErrProjectNotFound = errors.New("project not found")

	// ErrProjectAlreadyExists is returned when a project with the same repository URL already exists for a user
	ErrProjectAlreadyExists = errors.New("project with this repository URL already exists")

	// ErrUnauthorized is returned when a user tries to access a project they don't own
	ErrUnauthorized = errors.New("unauthorized to access this project")
)
