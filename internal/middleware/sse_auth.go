package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// SSEAuthMiddleware handles authentication for SSE endpoints
// Accepts token from either Authorization header or query parameter
func (m *AuthMiddleware) SSEAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get token from header first
		token := c.GetHeader("Authorization")

		// If not in header, check query parameter (for EventSource compatibility)
		if token == "" {
			token = c.Query("token")
			if token != "" {
				// Add Bearer prefix if not present
				if !strings.HasPrefix(token, "Bearer ") {
					token = "Bearer " + token
				}
			}
		}

		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "No authentication token provided",
			})
			c.Abort()
			return
		}

		// Verify token (same as RequireAuth)
		clerkUser, err := m.verifyToken(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "Invalid or expired token",
				"details": err.Error(),
			})
			c.Abort()
			return
		}

		// Set user in context
		c.Set("user", clerkUser)

		c.Next()
	}
}
