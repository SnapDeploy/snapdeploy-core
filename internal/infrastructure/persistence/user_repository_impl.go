package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"snapdeploy-core/internal/database"
	"snapdeploy-core/internal/domain/user"
)

// UserRepositoryImpl implements the domain user.Repository interface
type UserRepositoryImpl struct {
	db *database.DB
}

// NewUserRepository creates a new user repository implementation
func NewUserRepository(db *database.DB) user.Repository {
	return &UserRepositoryImpl{db: db}
}

// Save persists a user (create or update)
func (r *UserRepositoryImpl) Save(ctx context.Context, usr *user.User) error {
	queries := database.New(r.db.GetConnection())

	// Check if user exists
	_, err := queries.GetUserByID(ctx, usr.ID().UUID())
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to check if user exists: %w", err)
	}

	// If no error, user exists - update it
	if err == nil {
		// Update existing user
		_, err := queries.UpdateUser(ctx, &database.UpdateUserParams{
			Email:    usr.Email().String(),
			Username: usr.Username().String(),
			ID:       usr.ID().UUID(),
		})
		if err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}
	} else {
		// User doesn't exist (err == sql.ErrNoRows) - create it
		_, err := queries.CreateUser(ctx, &database.CreateUserParams{
			ID:          usr.ID().UUID(),
			Email:       usr.Email().String(),
			Username:    usr.Username().String(),
			ClerkUserID: usr.ClerkUserID().String(),
		})
		if err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}
	}

	return nil
}

// FindByID retrieves a user by their ID
func (r *UserRepositoryImpl) FindByID(ctx context.Context, id user.UserID) (*user.User, error) {
	queries := database.New(r.db.GetConnection())

	dbUser, err := queries.GetUserByID(ctx, id.UUID())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, user.ErrUserNotFound(id.String())
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return r.toDomain(dbUser)
}

// FindByEmail retrieves a user by their email
func (r *UserRepositoryImpl) FindByEmail(ctx context.Context, email user.Email) (*user.User, error) {
	queries := database.New(r.db.GetConnection())

	dbUser, err := queries.GetUserByEmail(ctx, email.String())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, user.ErrUserNotFound(email.String())
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return r.toDomain(dbUser)
}

// FindByClerkID retrieves a user by their Clerk user ID
func (r *UserRepositoryImpl) FindByClerkID(ctx context.Context, clerkID user.ClerkUserID) (*user.User, error) {
	queries := database.New(r.db.GetConnection())

	dbUser, err := queries.GetUserByClerkID(ctx, clerkID.String())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, user.ErrUserNotFound(clerkID.String())
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return r.toDomain(dbUser)
}

// Delete removes a user from persistence
func (r *UserRepositoryImpl) Delete(ctx context.Context, id user.UserID) error {
	queries := database.New(r.db.GetConnection())

	err := queries.DeleteUser(ctx, id.UUID())
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// List retrieves users with pagination
func (r *UserRepositoryImpl) List(ctx context.Context, limit, offset int32) ([]*user.User, error) {
	queries := database.New(r.db.GetConnection())

	dbUsers, err := queries.ListUsers(ctx, &database.ListUsersParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	users := make([]*user.User, len(dbUsers))
	for i, dbUser := range dbUsers {
		domainUser, err := r.toDomain(dbUser)
		if err != nil {
			return nil, fmt.Errorf("failed to convert user: %w", err)
		}
		users[i] = domainUser
	}

	return users, nil
}

// Count returns the total number of users
func (r *UserRepositoryImpl) Count(ctx context.Context) (int64, error) {
	queries := database.New(r.db.GetConnection())

	count, err := queries.CountUsers(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	return count, nil
}

// ExistsByEmail checks if a user with the given email exists
func (r *UserRepositoryImpl) ExistsByEmail(ctx context.Context, email user.Email) (bool, error) {
	queries := database.New(r.db.GetConnection())

	_, err := queries.GetUserByEmail(ctx, email.String())
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}

	return true, nil
}

// toDomain converts database user to domain user
func (r *UserRepositoryImpl) toDomain(dbUser *database.User) (*user.User, error) {
	var createdAt, updatedAt = dbUser.CreatedAt.Time, dbUser.UpdatedAt.Time

	return user.Reconstitute(
		dbUser.ID.String(),
		dbUser.Email,
		dbUser.Username,
		dbUser.ClerkUserID,
		createdAt,
		updatedAt,
	)
}
