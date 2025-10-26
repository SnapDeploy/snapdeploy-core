package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client handles GitHub API interactions
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new GitHub API client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://api.github.com",
	}
}

// Repository represents a GitHub repository from the API
type Repository struct {
	ID              int64   `json:"id"`
	Name            string  `json:"name"`
	FullName        string  `json:"full_name"`
	Description     *string `json:"description"`
	URL             string  `json:"url"`
	HTMLURL         string  `json:"html_url"`
	Private         bool    `json:"private"`
	Fork            bool    `json:"fork"`
	StargazersCount int32   `json:"stargazers_count"`
	WatchersCount   int32   `json:"watchers_count"`
	ForksCount      int32   `json:"forks_count"`
	DefaultBranch   string  `json:"default_branch"`
	Language        *string `json:"language"`
}

// GetUserRepositories fetches repositories for a user using their GitHub access token
func (c *Client) GetUserRepositories(ctx context.Context, accessToken string) ([]Repository, error) {
	url := fmt.Sprintf("%s/user/repos?per_page=100&sort=updated", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repositories: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github API returned status %d: %s", resp.StatusCode, string(body))
	}

	var repos []Repository
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, fmt.Errorf("failed to decode repositories: %w", err)
	}

	return repos, nil
}
