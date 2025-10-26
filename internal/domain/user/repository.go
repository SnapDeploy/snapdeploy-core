package user

import (
	"context"
)

// Repository defines the interface for user persistence
// This is defined in the domain layer, but implemented in infrastructure
type Repository interface {
	// Save persists a user (create or update)
	Save(ctx context.Context, user *User) error

	// FindByID retrieves a user by their ID
	FindByID(ctx context.Context, id UserID) (*User, error)

	// FindByEmail retrieves a user by their email
	FindByEmail(ctx context.Context, email Email) (*User, error)

	// FindByClerkID retrieves a user by their Clerk user ID
	FindByClerkID(ctx context.Context, clerkID ClerkUserID) (*User, error)

	// Delete removes a user from persistence
	Delete(ctx context.Context, id UserID) error

	// List retrieves users with pagination
	List(ctx context.Context, limit, offset int32) ([]*User, error)

	// Count returns the total number of users
	Count(ctx context.Context) (int64, error)

	// ExistsByEmail checks if a user with the given email exists
	ExistsByEmail(ctx context.Context, email Email) (bool, error)
}
