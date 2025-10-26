package github

import (
	"context"
	"fmt"

	"snapdeploy-core/internal/domain/repo"
	"snapdeploy-core/internal/github"
)

// GitHubServiceImpl implements the domain repo.GitHubService interface
type GitHubServiceImpl struct {
	client *github.Client
}

// NewGitHubService creates a new GitHub service implementation
func NewGitHubService(client *github.Client) repo.GitHubService {
	return &GitHubServiceImpl{client: client}
}

// FetchUserRepositories fetches all repositories for a user from GitHub
func (g *GitHubServiceImpl) FetchUserRepositories(ctx context.Context, accessToken string) ([]*repo.GitHubRepository, error) {
	// Use existing GitHub client
	githubRepos, err := g.client.GetUserRepositories(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repositories from GitHub: %w", err)
	}

	// Convert to domain GitHub repositories
	domainRepos := make([]*repo.GitHubRepository, len(githubRepos))
	for i, ghRepo := range githubRepos {
		domainRepos[i] = &repo.GitHubRepository{
			ID:              ghRepo.ID,
			Name:            ghRepo.Name,
			FullName:        ghRepo.FullName,
			Description:     ghRepo.Description,
			URL:             ghRepo.URL,
			HTMLURL:         ghRepo.HTMLURL,
			Private:         ghRepo.Private,
			Fork:            ghRepo.Fork,
			StargazersCount: ghRepo.StargazersCount,
			WatchersCount:   ghRepo.WatchersCount,
			ForksCount:      ghRepo.ForksCount,
			DefaultBranch:   ghRepo.DefaultBranch,
			Language:        ghRepo.Language,
		}
	}

	return domainRepos, nil
}
