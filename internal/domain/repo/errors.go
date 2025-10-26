package repo

import "fmt"

// Domain errors

type DomainError struct {
	Code    string
	Message string
	Err     error
}

func (e *DomainError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *DomainError) Unwrap() error {
	return e.Err
}

// Predefined domain errors

func ErrRepositoryNotFound(id string) *DomainError {
	return &DomainError{
		Code:    "REPOSITORY_NOT_FOUND",
		Message: fmt.Sprintf("repository with ID %s not found", id),
	}
}

func ErrInvalidRepositoryData(field string, err error) *DomainError {
	return &DomainError{
		Code:    "INVALID_REPOSITORY_DATA",
		Message: fmt.Sprintf("invalid %s", field),
		Err:     err,
	}
}

func ErrUnauthorizedAccess(userID, repoID string) *DomainError {
	return &DomainError{
		Code:    "UNAUTHORIZED_ACCESS",
		Message: fmt.Sprintf("user %s not authorized to access repository %s", userID, repoID),
	}
}
