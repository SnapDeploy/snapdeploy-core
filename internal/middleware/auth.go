package middleware

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"snapdeploy-core/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWK represents a JSON Web Key
type JWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// JWKSet represents a set of JSON Web Keys
type JWKSet struct {
	Keys []JWK `json:"keys"`
}

// AuthMiddleware handles JWT authentication using Clerk
type AuthMiddleware struct {
	jwksURL    string
	issuer     string
	publicKeys map[string]*rsa.PublicKey
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(cfg *config.Config) (*AuthMiddleware, error) {
	am := &AuthMiddleware{
		jwksURL:    cfg.Clerk.JWKSURL,
		issuer:     cfg.Clerk.Issuer,
		publicKeys: make(map[string]*rsa.PublicKey),
	}

	// Load public keys from JWKS endpoint
	if err := am.loadPublicKeys(); err != nil {
		return nil, fmt.Errorf("failed to load public keys: %w", err)
	}

	return am, nil
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

		// Verify the token with Clerk
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

// verifyToken verifies the JWT token with Clerk
func (am *AuthMiddleware) verifyToken(ctx context.Context, token string) (*ClerkUser, error) {
	// Parse the token to get the key ID
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		// Check the signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get the key ID from the token header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("missing key ID in token header")
		}

		// Get the public key for this key ID
		publicKey, exists := am.publicKeys[kid]
		if !exists {
			return nil, fmt.Errorf("unknown key ID: %s", kid)
		}

		return publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Check if the token is valid
	if !parsedToken.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Get the claims
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Verify the issuer
	issuer, ok := claims["iss"].(string)
	if !ok || issuer != am.issuer {
		return nil, fmt.Errorf("invalid issuer")
	}

	// Extract user information from claims
	user := &ClerkUser{}

	if sub, ok := claims["sub"].(string); ok {
		user.ID = sub
	}
	if email, ok := claims["email"].(string); ok {
		user.Email = email
	}
	if username, ok := claims["username"].(string); ok {
		user.Username = username
	}
	if firstName, ok := claims["given_name"].(string); ok {
		user.FirstName = firstName
	}
	if lastName, ok := claims["family_name"].(string); ok {
		user.LastName = lastName
	}

	return user, nil
}

// loadPublicKeys loads public keys from the JWKS endpoint
func (am *AuthMiddleware) loadPublicKeys() error {
	resp, err := http.Get(am.jwksURL)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	var jwks JWKSet
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("failed to decode JWKS: %w", err)
	}

	// Convert JWKs to RSA public keys
	for _, jwk := range jwks.Keys {
		if jwk.Kty != "RSA" {
			continue
		}

		// Decode the modulus and exponent
		nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
		if err != nil {
			continue
		}

		eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
		if err != nil {
			continue
		}

		// Create the RSA public key
		publicKey := &rsa.PublicKey{
			N: new(big.Int).SetBytes(nBytes),
			E: int(new(big.Int).SetBytes(eBytes).Int64()),
		}

		am.publicKeys[jwk.Kid] = publicKey
	}

	return nil
}

// ClerkUser represents a user from Clerk
type ClerkUser struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// GetUserID returns the user ID
func (u *ClerkUser) GetUserID() string {
	return u.ID
}

// GetEmail returns the user's email
func (u *ClerkUser) GetEmail() string {
	return u.Email
}

// GetUsername returns the user's username
func (u *ClerkUser) GetUsername() string {
	return u.Username
}
