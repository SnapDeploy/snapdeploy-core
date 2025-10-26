package user

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

func ErrUserNotFound(id string) *DomainError {
	return &DomainError{
		Code:    "USER_NOT_FOUND",
		Message: fmt.Sprintf("user with ID %s not found", id),
	}
}

func ErrUserAlreadyExists(email string) *DomainError {
	return &DomainError{
		Code:    "USER_ALREADY_EXISTS",
		Message: fmt.Sprintf("user with email %s already exists", email),
	}
}

func ErrInvalidUserData(field string, err error) *DomainError {
	return &DomainError{
		Code:    "INVALID_USER_DATA",
		Message: fmt.Sprintf("invalid %s", field),
		Err:     err,
	}
}
