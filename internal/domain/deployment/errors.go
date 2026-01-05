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

	// ErrCannotExtendNonDeployed is returned when trying to extend a deployment that is not in DEPLOYED status
	ErrCannotExtendNonDeployed = errors.New("can only extend deployments in DEPLOYED status")

	// ErrMaxExtensionsReached is returned when the deployment has been extended the maximum number of times
	ErrMaxExtensionsReached = errors.New("maximum number of extensions reached")

	// ErrDeploymentExpired is returned when trying to perform an operation on an expired deployment
	ErrDeploymentExpired = errors.New("deployment has expired")

	// ErrActiveDeploymentExists is returned when trying to create a deployment while one is already active
	ErrActiveDeploymentExists = errors.New("active deployment already exists for this project")
)

