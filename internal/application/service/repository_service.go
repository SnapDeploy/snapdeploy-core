package service

import (
	"context"
	"fmt"
	"time"

	"snapdeploy-core/internal/application/dto"
	"snapdeploy-core/internal/domain/repo"
	"snapdeploy-core/internal/domain/user"
)

// RepositoryService handles repository-related use cases
type RepositoryService struct {
	repoRepo      repo.RepositoryRepo
	githubService repo.GitHubService
}

// NewRepositoryService creates a new repository service
func NewRepositoryService(repoRepo repo.RepositoryRepo, githubService repo.GitHubService) *RepositoryService {
	return &RepositoryService{
		repoRepo:      repoRepo,
		githubService: githubService,
	}
}

// SyncRepositoriesFromGitHub fetches repositories from GitHub and syncs them
func (s *RepositoryService) SyncRepositoriesFromGitHub(ctx context.Context, userID string, githubAccessToken string) (*dto.RepositorySyncResponse, error) {
	// Parse user ID
	uid, err := user.ParseUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Fetch repositories from GitHub
	githubRepos, err := s.githubService.FetchUserRepositories(ctx, githubAccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repositories from GitHub: %w", err)
	}

	var repositories []*repo.Repository

	// Process each GitHub repository
	for _, ghRepo := range githubRepos {
		// Try to find existing repository by URL
		repoURL, err := repo.NewURL(ghRepo.URL)
		if err != nil {
			continue // Skip invalid URLs
		}

		existingRepo, err := s.repoRepo.FindByURL(ctx, repoURL)
		if err == nil {
			// Update existing repository
			existingRepo.UpdateMetadata(
				ghRepo.Description,
				&ghRepo.HTMLURL,
				ghRepo.Private,
				ghRepo.Fork,
				ghRepo.StargazersCount,
				ghRepo.WatchersCount,
				ghRepo.ForksCount,
				&ghRepo.DefaultBranch,
				ghRepo.Language,
			)
			if err := s.repoRepo.Save(ctx, existingRepo); err != nil {
				return nil, fmt.Errorf("failed to update repository: %w", err)
			}
			repositories = append(repositories, existingRepo)
		} else {
			// Create new repository
			newRepo, err := repo.NewRepository(
				uid,
				ghRepo.ID,
				ghRepo.Name,
				ghRepo.FullName,
				ghRepo.URL,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to create repository entity: %w", err)
			}

			// Update metadata
			newRepo.UpdateMetadata(
				ghRepo.Description,
				&ghRepo.HTMLURL,
				ghRepo.Private,
				ghRepo.Fork,
				ghRepo.StargazersCount,
				ghRepo.WatchersCount,
				ghRepo.ForksCount,
				&ghRepo.DefaultBranch,
				ghRepo.Language,
			)

			if err := s.repoRepo.Save(ctx, newRepo); err != nil {
				return nil, fmt.Errorf("failed to save repository: %w", err)
			}
			repositories = append(repositories, newRepo)
		}
	}

	return &dto.RepositorySyncResponse{
		Message: "success",
	}, nil
}

// GetRepositoriesByUserID retrieves repositories for a user with pagination and optional search
func (s *RepositoryService) GetRepositoriesByUserID(ctx context.Context, userID string, searchQuery string, page, limit int32) (*dto.RepositoryListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	uid, err := user.ParseUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	offset := (page - 1) * limit

	var repositories []*repo.Repository
	var total int64

	// Use search if query provided
	if searchQuery != "" {
		repositories, err = s.repoRepo.SearchByUserID(ctx, uid, searchQuery, limit, offset)
		if err != nil {
			return nil, fmt.Errorf("failed to search repositories: %w", err)
		}

		total, err = s.repoRepo.CountSearchByUserID(ctx, uid, searchQuery)
		if err != nil {
			return nil, fmt.Errorf("failed to count search repositories: %w", err)
		}
	} else {
		repositories, err = s.repoRepo.FindByUserID(ctx, uid, limit, offset)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch repositories: %w", err)
		}

		total, err = s.repoRepo.CountByUserID(ctx, uid)
		if err != nil {
			return nil, fmt.Errorf("failed to count repositories: %w", err)
		}
	}

	repoResponses := make([]*dto.RepositoryResponse, len(repositories))
	for i, repository := range repositories {
		repoResponses[i] = s.toDTO(repository)
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	return &dto.RepositoryListResponse{
		Repositories: repoResponses,
		Pagination: dto.PaginationResponse{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// toDTO converts a domain repository to DTO
func (s *RepositoryService) toDTO(r *repo.Repository) *dto.RepositoryResponse {
	return &dto.RepositoryResponse{
		ID:          r.ID().String(),
		Name:        r.Name().String(),
		FullName:    r.FullName(),
		Description: r.Description(),
		URL:         r.URL().String(),
		HTMLURL:     r.HTMLURL(),
		Private:     r.IsPrivate(),
		Fork:        r.IsFork(),
		Stars:       r.StargazersCount(),
		Watchers:    r.WatchersCount(),
		Forks:       r.ForksCount(),
		Language:    r.Language(),
		CreatedAt:   r.CreatedAt().Format(time.RFC3339),
	}
}
