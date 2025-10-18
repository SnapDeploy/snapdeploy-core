package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/gin-gonic/gin"
	"snapdeploy-core/internal/config"
)

// AuthMiddleware handles JWT authentication using AWS Cognito
type AuthMiddleware struct {
	cognitoClient *cognitoidentityprovider.Client
	userPoolID    string
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(cfg *config.Config) (*AuthMiddleware, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithRegion(cfg.AWS.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	cognitoClient := cognitoidentityprovider.NewFromConfig(awsCfg)

	return &AuthMiddleware{
		cognitoClient: cognitoClient,
		userPoolID:    cfg.AWS.CognitoUserPool,
	}, nil
}

// RequireAuth is a Gin middleware that requires authentication
func (am *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Authorization header is required",
			})
			c.Abort()
			return
		}

		// Check if the header starts with "Bearer "
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Authorization header must start with 'Bearer '",
			})
			c.Abort()
			return
		}

		// Extract the token
		token := strings.TrimPrefix(authHeader, "Bearer ")

		// Verify the token with Cognito
		user, err := am.verifyToken(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Invalid token",
				"details": err.Error(),
			})
			c.Abort()
			return
		}

		// Store user information in context
		c.Set("user", user)
		c.Next()
	}
}

// verifyToken verifies the JWT token with AWS Cognito
func (am *AuthMiddleware) verifyToken(ctx context.Context, token string) (*CognitoUser, error) {
	input := &cognitoidentityprovider.GetUserInput{
		AccessToken: aws.String(token),
	}

	result, err := am.cognitoClient.GetUser(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to verify token: %w", err)
	}

	// Extract user information from Cognito response
	user := &CognitoUser{
		Username: *result.Username,
	}

	// Extract attributes
	for _, attr := range result.UserAttributes {
		switch *attr.Name {
		case "email":
			user.Email = *attr.Value
		case "sub":
			user.Sub = *attr.Value
		case "given_name":
			user.GivenName = *attr.Value
		case "family_name":
			user.FamilyName = *attr.Value
		}
	}

	return user, nil
}

// CognitoUser represents a user from AWS Cognito
type CognitoUser struct {
	Username   string `json:"username"`
	Email      string `json:"email"`
	Sub        string `json:"sub"`
	GivenName  string `json:"given_name"`
	FamilyName string `json:"family_name"`
}

// GetUserID returns the user ID (sub claim)
func (u *CognitoUser) GetUserID() string {
	return u.Sub
}

// GetEmail returns the user's email
func (u *CognitoUser) GetEmail() string {
	return u.Email
}

// GetUsername returns the user's username
func (u *CognitoUser) GetUsername() string {
	return u.Username
}
