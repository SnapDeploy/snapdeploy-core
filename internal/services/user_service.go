package services

import (
	"context"
	"fmt"

	"snapdeploy-core/internal/middleware"
	"snapdeploy-core/internal/models"
	"snapdeploy-core/internal/repositories"
	"github.com/google/uuid"
)

// UserService handles user business logic
type UserService struct {
	userRepo *repositories.UserRepository
}

// NewUserService creates a new user service
func NewUserService(userRepo *repositories.UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

// CreateUser creates a new user
func (s *UserService) CreateUser(ctx context.Context, req *models.CreateUserRequest) (*models.User, error) {
	// Check if user already exists by email
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("user with email %s already exists", req.Email)
	}

	// Check if user already exists by Cognito ID
	existingUser, err = s.userRepo.GetByCognitoID(ctx, req.CognitoUserID)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("user with Cognito ID %s already exists", req.CognitoUserID)
	}

	// Create new user
	user := &models.User{
		ID:            generateID(), // You'll need to implement this
		Email:         req.Email,
		Username:      req.Username,
		CognitoUserID: req.CognitoUserID,
	}

	return s.userRepo.Create(ctx, user)
}

// GetUserByID retrieves a user by ID
func (s *UserService) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

// GetUserByEmail retrieves a user by email
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	return s.userRepo.GetByEmail(ctx, email)
}

// GetUserByCognitoID retrieves a user by Cognito user ID
func (s *UserService) GetUserByCognitoID(ctx context.Context, cognitoUserID string) (*models.User, error) {
	return s.userRepo.GetByCognitoID(ctx, cognitoUserID)
}

// UpdateUser updates a user
func (s *UserService) UpdateUser(ctx context.Context, id string, req *models.UpdateUserRequest) (*models.User, error) {
	// Get existing user
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.Email != nil {
		// Check if email is already taken by another user
		existingUser, err := s.userRepo.GetByEmail(ctx, *req.Email)
		if err == nil && existingUser != nil && existingUser.ID != id {
			return nil, fmt.Errorf("email %s is already taken", *req.Email)
		}
		user.Email = *req.Email
	}

	if req.Username != nil {
		user.Username = *req.Username
	}

	return s.userRepo.Update(ctx, user)
}

// DeleteUser deletes a user
func (s *UserService) DeleteUser(ctx context.Context, id string) error {
	// Check if user exists
	_, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	return s.userRepo.Delete(ctx, id)
}

// ListUsers retrieves a list of users with pagination
func (s *UserService) ListUsers(ctx context.Context, page, limit int32) ([]*models.User, int64, error) {
	offset := (page - 1) * limit

	users, err := s.userRepo.List(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.userRepo.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// GetOrCreateUserByCognitoID gets an existing user or creates a new one based on Cognito user ID
func (s *UserService) GetOrCreateUserByCognitoID(ctx context.Context, cognitoUser *middleware.CognitoUser) (*models.User, error) {
	// Try to get existing user
	user, err := s.userRepo.GetByCognitoID(ctx, cognitoUser.GetUserID())
	if err == nil {
		return user, nil
	}

	// User doesn't exist, create new one
	createReq := &models.CreateUserRequest{
		Email:         cognitoUser.GetEmail(),
		Username:      cognitoUser.GetUsername(),
		CognitoUserID: cognitoUser.GetUserID(),
	}

	return s.CreateUser(ctx, createReq)
}

// generateID generates a new UUID
func generateID() string {
	return uuid.New().String()
}
