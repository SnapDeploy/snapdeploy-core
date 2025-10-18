package clerk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"snapdeploy-core/internal/config"
)

// Client represents a Clerk API client
type Client struct {
	apiURL     string
	secretKey  string
	httpClient *http.Client
}

// NewClient creates a new Clerk API client
func NewClient(cfg *config.ClerkConfig) *Client {
	return &Client{
		apiURL:    cfg.APIURL,
		secretKey: cfg.SecretKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// User represents a user from Clerk API
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email_address"`
	Username  string    `json:"username"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// EmailAddress represents an email address from Clerk API
type EmailAddress struct {
	ID           string `json:"id"`
	EmailAddress string `json:"email_address"`
	Reserved     bool   `json:"reserved"`
}

// ClerkUserResponse represents the full response from Clerk API
type ClerkUserResponse struct {
	ID             string         `json:"id"`
	Object         string         `json:"object"`
	Username       string         `json:"username"`
	FirstName      string         `json:"first_name"`
	LastName       string         `json:"last_name"`
	EmailAddresses []EmailAddress `json:"email_addresses"`
	CreatedAt      int64          `json:"created_at"`
	UpdatedAt      int64          `json:"updated_at"`
}

// GetUserResponse represents the response from Clerk API
type GetUserResponse struct {
	User ClerkUserResponse `json:"data"`
}

// DirectUserResponse represents the direct response from Clerk API (without data wrapper)
type DirectUserResponse struct {
	ID             string         `json:"id"`
	Object         string         `json:"object"`
	Username       string         `json:"username"`
	FirstName      string         `json:"first_name"`
	LastName       string         `json:"last_name"`
	EmailAddresses []EmailAddress `json:"email_addresses"`
	CreatedAt      int64          `json:"created_at"`
	UpdatedAt      int64          `json:"updated_at"`
}

// GetUser fetches a user by ID from Clerk API
func (c *Client) GetUser(ctx context.Context, userID string) (*User, error) {
	url := fmt.Sprintf("%s/users/%s", c.apiURL, userID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.secretKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("clerk API error: %d - %s", resp.StatusCode, string(body))
	}

	// Try to parse as direct response first (without data wrapper)
	var directResp DirectUserResponse
	if err := json.Unmarshal(body, &directResp); err != nil {
		// Try parsing as wrapped response
		var userResp GetUserResponse
		if err := json.Unmarshal(body, &userResp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}
		// Convert ClerkUserResponse to DirectUserResponse
		directResp = DirectUserResponse{
			ID:             userResp.User.ID,
			Object:         userResp.User.Object,
			Username:       userResp.User.Username,
			FirstName:      userResp.User.FirstName,
			LastName:       userResp.User.LastName,
			EmailAddresses: userResp.User.EmailAddresses,
			CreatedAt:      userResp.User.CreatedAt,
			UpdatedAt:      userResp.User.UpdatedAt,
		}
	}

	// Extract email from email_addresses array
	email := ""
	if len(directResp.EmailAddresses) > 0 {
		email = directResp.EmailAddresses[0].EmailAddress
	}

	// Convert timestamps to time.Time
	createdAt := time.Unix(directResp.CreatedAt/1000, 0)
	updatedAt := time.Unix(directResp.UpdatedAt/1000, 0)

	user := &User{
		ID:        directResp.ID,
		Email:     email,
		Username:  directResp.Username,
		FirstName: directResp.FirstName,
		LastName:  directResp.LastName,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	return user, nil
}

// GetUserByEmail fetches a user by email from Clerk API
func (c *Client) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	url := fmt.Sprintf("%s/users?email_address=%s", c.apiURL, email)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.secretKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("clerk API error: %d - %s", resp.StatusCode, string(body))
	}

	var usersResp struct {
		Data []User `json:"data"`
	}
	if err := json.Unmarshal(body, &usersResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(usersResp.Data) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	return &usersResp.Data[0], nil
}
