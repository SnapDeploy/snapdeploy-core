package repo

import (
	"context"
)

// GitHubRepository represents a repository fetched from GitHub API
type GitHubRepository struct {
	ID              int64
	Name            string
	FullName        string
	Description     *string
	URL             string
	HTMLURL         string
	Private         bool
	Fork            bool
	StargazersCount int32
	WatchersCount   int32
	ForksCount      int32
	DefaultBranch   string
	Language        *string
}

// GitHubService is a domain service interface for interacting with GitHub
// Implementation will be in infrastructure layer
type GitHubService interface {
	// FetchUserRepositories fetches all repositories for a user from GitHub
	FetchUserRepositories(ctx context.Context, accessToken string) ([]*GitHubRepository, error)
}
