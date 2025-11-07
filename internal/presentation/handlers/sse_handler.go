package handlers

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// SSEClient represents a connected SSE client
type SSEClient struct {
	ID           string
	DeploymentID string
	Channel      chan string
	Context      context.Context
}

// SSEManager manages SSE connections
type SSEManager struct {
	clients map[string][]*SSEClient // deploymentID -> clients
	mu      sync.RWMutex
}

// NewSSEManager creates a new SSE manager
func NewSSEManager() *SSEManager {
	return &SSEManager{
		clients: make(map[string][]*SSEClient),
	}
}

// AddClient registers a new SSE client
func (m *SSEManager) AddClient(deploymentID string, client *SSEClient) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.clients[deploymentID] == nil {
		m.clients[deploymentID] = make([]*SSEClient, 0)
	}
	m.clients[deploymentID] = append(m.clients[deploymentID], client)
}

// RemoveClient removes an SSE client
func (m *SSEManager) RemoveClient(deploymentID string, clientID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	clients := m.clients[deploymentID]
	for i, client := range clients {
		if client.ID == clientID {
			close(client.Channel)
			m.clients[deploymentID] = append(clients[:i], clients[i+1:]...)
			break
		}
	}

	// Clean up empty deployment entries
	if len(m.clients[deploymentID]) == 0 {
		delete(m.clients, deploymentID)
	}
}

// BroadcastLog sends a log line to all clients watching a deployment
func (m *SSEManager) BroadcastLog(deploymentID string, logLine string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clients := m.clients[deploymentID]
	for _, client := range clients {
		select {
		case client.Channel <- logLine:
			// Sent successfully
		case <-time.After(1 * time.Second):
			// Client is slow or disconnected, skip
		}
	}
}

// GetClientCount returns the number of clients watching a deployment
func (m *SSEManager) GetClientCount(deploymentID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients[deploymentID])
}

// StreamDeploymentLogs handles SSE streaming of deployment logs
// @Summary Stream deployment logs
// @Description Streams deployment logs in real-time using Server-Sent Events
// @Tags Deployments
// @Produce text/event-stream
// @Param id path string true "Deployment ID"
// @Param token query string false "Auth token (if not in header)"
// @Success 200 {string} string "SSE stream"
// @Router /deployments/{id}/logs/stream [get]
func (h *DeploymentHandler) StreamDeploymentLogs(c *gin.Context) {
	deploymentID := c.Param("id")

	// DEBUG: Log SSE request
	fmt.Printf("[SSE DEBUG] StreamDeploymentLogs called for deployment: %s\n", deploymentID)
	fmt.Printf("[SSE DEBUG] Request path: %s\n", c.Request.URL.Path)
	fmt.Printf("[SSE DEBUG] Request query: %s\n", c.Request.URL.RawQuery)
	fmt.Printf("[SSE DEBUG] Authorization header: %s\n", c.GetHeader("Authorization"))
	fmt.Printf("[SSE DEBUG] Token query param: %s\n", c.Query("token"))

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // Disable nginx buffering
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Credentials", "true")

	fmt.Printf("[SSE DEBUG] Headers set, creating SSE client\n")

	// Create client
	clientID := fmt.Sprintf("client_%d", time.Now().UnixNano())
	client := &SSEClient{
		ID:           clientID,
		DeploymentID: deploymentID,
		Channel:      make(chan string, 100),
		Context:      c.Request.Context(),
	}

	// Register client
	sseManager.AddClient(deploymentID, client)
	defer sseManager.RemoveClient(deploymentID, clientID)

	// Send existing logs when client first connects
	// This ensures clients connecting mid-deployment see all logs
	deployment, err := h.deploymentService.GetDeploymentByID(c.Request.Context(), deploymentID)
	if err == nil && deployment.Logs != "" {
		// Send existing logs line by line
		existingLines := strings.Split(deployment.Logs, "\n")
		for _, line := range existingLines {
			if line != "" {
				c.SSEvent("log", line)
			}
		}
		c.Writer.Flush()
	}

	// Stream new logs
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			// Client disconnected
			return
		case logLine := <-client.Channel:
			// Send log line via SSE
			c.SSEvent("log", logLine)
			c.Writer.Flush()
		case <-ticker.C:
			// Send heartbeat to keep connection alive
			c.SSEvent("heartbeat", "ping")
			c.Writer.Flush()
		}
	}
}

// Global SSE manager instance
var sseManager = NewSSEManager()

// GetSSEManager returns the global SSE manager
func GetSSEManager() *SSEManager {
	return sseManager
}
