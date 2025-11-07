package deployment

import "errors"

var (
	// ErrDeploymentNotFound is returned when a deployment is not found
	ErrDeploymentNotFound = errors.New("deployment not found")

	// ErrInvalidStatusTransition is returned when trying to transition to an invalid status
	ErrInvalidStatusTransition = errors.New("invalid deployment status transition")

	// ErrUnauthorized is returned when a user tries to access a deployment they don't own
	ErrUnauthorized = errors.New("unauthorized to access this deployment")

	// ErrProjectNotFound is returned when the associated project is not found
	ErrProjectNotFound = errors.New("project not found for deployment")
)

