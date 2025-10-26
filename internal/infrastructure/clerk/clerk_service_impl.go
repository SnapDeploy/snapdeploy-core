package clerk

import (
	"context"
	"fmt"

	"snapdeploy-core/internal/application/service"
	"snapdeploy-core/internal/clerk"
)

// ClerkServiceImpl implements the application service.ClerkService interface
type ClerkServiceImpl struct {
	client *clerk.Client
}

// NewClerkService creates a new Clerk service implementation
func NewClerkService(client *clerk.Client) service.ClerkService {
	return &ClerkServiceImpl{client: client}
}

// GetUser fetches user data from Clerk
func (c *ClerkServiceImpl) GetUser(ctx context.Context, clerkUserID string) (*service.ClerkUserData, error) {
	user, err := c.client.GetUser(ctx, clerkUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user from Clerk: %w", err)
	}

	return &service.ClerkUserData{
		ID:       user.ID,
		Email:    user.Email,
		Username: user.Username,
	}, nil
}
