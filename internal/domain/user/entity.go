package user

import (
	"fmt"
	"time"
)

// User is a domain entity representing a user in the system
type User struct {
	id          UserID
	email       Email
	username    Username
	clerkUserID ClerkUserID
	createdAt   time.Time
	updatedAt   time.Time
}

// NewUser creates a new User entity with validation
func NewUser(email, username, clerkUserID string) (*User, error) {
	emailVO, err := NewEmail(email)
	if err != nil {
		return nil, fmt.Errorf("invalid email: %w", err)
	}

	usernameVO, err := NewUsername(username)
	if err != nil {
		return nil, fmt.Errorf("invalid username: %w", err)
	}

	clerkIDVO, err := NewClerkUserID(clerkUserID)
	if err != nil {
		return nil, fmt.Errorf("invalid clerk user ID: %w", err)
	}

	now := time.Now()
	return &User{
		id:          NewUserID(),
		email:       emailVO,
		username:    usernameVO,
		clerkUserID: clerkIDVO,
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

// Reconstitute recreates a User entity from persistence
func Reconstitute(id, email, username, clerkUserID string, createdAt, updatedAt time.Time) (*User, error) {
	userID, err := ParseUserID(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	emailVO, err := NewEmail(email)
	if err != nil {
		return nil, fmt.Errorf("invalid email: %w", err)
	}

	usernameVO, err := NewUsername(username)
	if err != nil {
		return nil, fmt.Errorf("invalid username: %w", err)
	}

	clerkIDVO, err := NewClerkUserID(clerkUserID)
	if err != nil {
		return nil, fmt.Errorf("invalid clerk user ID: %w", err)
	}

	return &User{
		id:          userID,
		email:       emailVO,
		username:    usernameVO,
		clerkUserID: clerkIDVO,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}, nil
}

// UpdateEmail updates the user's email
func (u *User) UpdateEmail(newEmail string) error {
	emailVO, err := NewEmail(newEmail)
	if err != nil {
		return fmt.Errorf("invalid email: %w", err)
	}

	u.email = emailVO
	u.updatedAt = time.Now()
	return nil
}

// UpdateUsername updates the user's username
func (u *User) UpdateUsername(newUsername string) error {
	usernameVO, err := NewUsername(newUsername)
	if err != nil {
		return fmt.Errorf("invalid username: %w", err)
	}

	u.username = usernameVO
	u.updatedAt = time.Now()
	return nil
}

// Getters

func (u *User) ID() UserID {
	return u.id
}

func (u *User) Email() Email {
	return u.email
}

func (u *User) Username() Username {
	return u.username
}

func (u *User) ClerkUserID() ClerkUserID {
	return u.clerkUserID
}

func (u *User) CreatedAt() time.Time {
	return u.createdAt
}

func (u *User) UpdatedAt() time.Time {
	return u.updatedAt
}

// String returns string representation (for debugging)
func (u *User) String() string {
	return fmt.Sprintf("User{id: %s, email: %s, username: %s}",
		u.id.String(), u.email.String(), u.username.String())
}
