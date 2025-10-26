package service

import (
	"context"
	"fmt"

	"snapdeploy-core/internal/application/dto"
	"snapdeploy-core/internal/domain/repo"
	"snapdeploy-core/internal/domain/user"
)

// ClerkUserData represents user data fetched from Clerk
type ClerkUserData struct {
	ID       string
	Email    string
	Username string
}

// ClerkService is an interface for interacting with Clerk
type ClerkService interface {
	GetUser(ctx context.Context, clerkUserID string) (*ClerkUserData, error)
}

// UserService handles user-related use cases
type UserService struct {
	userRepo    user.Repository
	repoRepo    repo.RepositoryRepo
	clerkClient ClerkService
}

// NewUserService creates a new user service
func NewUserService(userRepo user.Repository, repoRepo repo.RepositoryRepo, clerkClient ClerkService) *UserService {
	return &UserService{
		userRepo:    userRepo,
		repoRepo:    repoRepo,
		clerkClient: clerkClient,
	}
}

// CreateUser creates a new user
func (s *UserService) CreateUser(ctx context.Context, req *dto.CreateUserRequest) (*dto.UserResponse, error) {
	// Check if user already exists by email
	email, err := user.NewEmail(req.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid email: %w", err)
	}

	exists, err := s.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to check if user exists: %w", err)
	}
	if exists {
		return nil, user.ErrUserAlreadyExists(req.Email)
	}

	// Create domain entity
	domainUser, err := user.NewUser(req.Email, req.Username, req.ClerkUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to create user entity: %w", err)
	}

	// Persist
	if err := s.userRepo.Save(ctx, domainUser); err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	return s.toDTO(ctx, domainUser), nil
}

// GetUserByID retrieves a user by ID
func (s *UserService) GetUserByID(ctx context.Context, id string) (*dto.UserResponse, error) {
	userID, err := user.ParseUserID(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	domainUser, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return s.toDTO(ctx, domainUser), nil
}

// GetUserByClerkID retrieves a user by Clerk ID
func (s *UserService) GetUserByClerkID(ctx context.Context, clerkUserID string) (*dto.UserResponse, error) {
	clerkID, err := user.NewClerkUserID(clerkUserID)
	if err != nil {
		return nil, fmt.Errorf("invalid clerk user ID: %w", err)
	}

	domainUser, err := s.userRepo.FindByClerkID(ctx, clerkID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return s.toDTO(ctx, domainUser), nil
}

// GetOrCreateUserByClerkID gets or creates a user based on Clerk ID
func (s *UserService) GetOrCreateUserByClerkID(ctx context.Context, clerkUserID string) (*dto.UserResponse, error) {
	clerkID, err := user.NewClerkUserID(clerkUserID)
	if err != nil {
		return nil, fmt.Errorf("invalid clerk user ID: %w", err)
	}

	// Try to find existing user
	domainUser, err := s.userRepo.FindByClerkID(ctx, clerkID)
	if err == nil {
		return s.toDTO(ctx, domainUser), nil
	}

	// User doesn't exist, fetch from Clerk and create
	clerkUserData, err := s.clerkClient.GetUser(ctx, clerkUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user from Clerk: %w", err)
	}

	// Create new user
	domainUser, err = user.NewUser(clerkUserData.Email, clerkUserData.Username, clerkUserData.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create user entity: %w", err)
	}

	if err := s.userRepo.Save(ctx, domainUser); err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	return s.toDTO(ctx, domainUser), nil
}

// UpdateUser updates a user
func (s *UserService) UpdateUser(ctx context.Context, id string, req *dto.UpdateUserRequest) (*dto.UserResponse, error) {
	userID, err := user.ParseUserID(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	domainUser, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Update fields if provided
	if req.Email != nil {
		// Check if email is already taken
		email, err := user.NewEmail(*req.Email)
		if err != nil {
			return nil, fmt.Errorf("invalid email: %w", err)
		}

		existingUser, err := s.userRepo.FindByEmail(ctx, email)
		if err == nil && !existingUser.ID().Equals(domainUser.ID()) {
			return nil, fmt.Errorf("email %s is already taken", *req.Email)
		}

		if err := domainUser.UpdateEmail(*req.Email); err != nil {
			return nil, fmt.Errorf("failed to update email: %w", err)
		}
	}

	if req.Username != nil {
		if err := domainUser.UpdateUsername(*req.Username); err != nil {
			return nil, fmt.Errorf("failed to update username: %w", err)
		}
	}

	if err := s.userRepo.Save(ctx, domainUser); err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	return s.toDTO(ctx, domainUser), nil
}

// DeleteUser deletes a user
func (s *UserService) DeleteUser(ctx context.Context, id string) error {
	userID, err := user.ParseUserID(id)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	// Check if user exists
	_, err = s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to find user: %w", err)
	}

	if err := s.userRepo.Delete(ctx, userID); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// ListUsers retrieves users with pagination
func (s *UserService) ListUsers(ctx context.Context, page, limit int32) (*dto.UserListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	users, err := s.userRepo.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	total, err := s.userRepo.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	userResponses := make([]*dto.UserResponse, len(users))
	for i, u := range users {
		userResponses[i] = s.toDTO(ctx, u)
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	return &dto.UserListResponse{
		Users: userResponses,
		Pagination: dto.PaginationResponse{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// toDTO converts a domain user to DTO
func (s *UserService) toDTO(ctx context.Context, u *user.User) *dto.UserResponse {
	// Check if user has synced repositories
	hasSyncedRepos := false
	count, err := s.repoRepo.CountByUserID(ctx, u.ID())
	if err == nil && count > 0 {
		hasSyncedRepos = true
	}

	return &dto.UserResponse{
		ID:                    u.ID().String(),
		Email:                 u.Email().String(),
		Username:              u.Username().String(),
		HasSyncedRepositories: hasSyncedRepos,
		CreatedAt:             u.CreatedAt(),
		UpdatedAt:             u.UpdatedAt(),
	}
}
