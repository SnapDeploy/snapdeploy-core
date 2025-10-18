package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"snapdeploy-core/internal/database"
	"snapdeploy-core/internal/models"
)

// UserRepository handles user data operations
type UserRepository struct {
	db *database.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *database.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *models.User) (*models.User, error) {
	queries := database.New(r.db.GetConnection())
	
	userID, err := uuid.Parse(user.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	
	dbUser, err := queries.CreateUser(ctx, &database.CreateUserParams{
		ID:            userID,
		Email:         user.Email,
		Username:      user.Username,
		CognitoUserID: user.CognitoUserID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return r.dbUserToModel(*dbUser), nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	queries := database.New(r.db.GetConnection())
	
	userID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	
	dbUser, err := queries.GetUserByID(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return r.dbUserToModel(*dbUser), nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	queries := database.New(r.db.GetConnection())
	
	dbUser, err := queries.GetUserByEmail(ctx, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return r.dbUserToModel(*dbUser), nil
}

// GetByCognitoID retrieves a user by Cognito user ID
func (r *UserRepository) GetByCognitoID(ctx context.Context, cognitoUserID string) (*models.User, error) {
	queries := database.New(r.db.GetConnection())
	
	dbUser, err := queries.GetUserByCognitoID(ctx, cognitoUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return r.dbUserToModel(*dbUser), nil
}

// Update updates a user
func (r *UserRepository) Update(ctx context.Context, user *models.User) (*models.User, error) {
	queries := database.New(r.db.GetConnection())
	
	userID, err := uuid.Parse(user.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	
	dbUser, err := queries.UpdateUser(ctx, &database.UpdateUserParams{
		Email:    user.Email,
		Username: user.Username,
		ID:       userID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return r.dbUserToModel(*dbUser), nil
}

// Delete deletes a user
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	queries := database.New(r.db.GetConnection())
	
	userID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}
	
	err = queries.DeleteUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// List retrieves a list of users with pagination
func (r *UserRepository) List(ctx context.Context, limit, offset int32) ([]*models.User, error) {
	queries := database.New(r.db.GetConnection())
	
	dbUsers, err := queries.ListUsers(ctx, &database.ListUsersParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	users := make([]*models.User, len(dbUsers))
	for i, dbUser := range dbUsers {
		users[i] = r.dbUserToModel(*dbUser)
	}

	return users, nil
}

// Count returns the total number of users
func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	queries := database.New(r.db.GetConnection())
	
	count, err := queries.CountUsers(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	return count, nil
}

// dbUserToModel converts a database user to a model user
func (r *UserRepository) dbUserToModel(dbUser database.User) *models.User {
	var createdAt, updatedAt time.Time
	if dbUser.CreatedAt.Valid {
		createdAt = dbUser.CreatedAt.Time
	}
	if dbUser.UpdatedAt.Valid {
		updatedAt = dbUser.UpdatedAt.Time
	}

	return &models.User{
		ID:            dbUser.ID.String(),
		Email:         dbUser.Email,
		Username:      dbUser.Username,
		CognitoUserID: dbUser.CognitoUserID,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
}
