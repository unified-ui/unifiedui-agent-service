// Package handlers provides HTTP handlers for the API.
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/unifiedui/chat-service/internal/core/cache"
	"github.com/unifiedui/chat-service/internal/core/docdb"
)

// HealthHandler handles health check endpoints.
type HealthHandler struct {
	cacheClient cache.Client
	docDBClient docdb.Client
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(cacheClient cache.Client, docDBClient docdb.Client) *HealthHandler {
	return &HealthHandler{
		cacheClient: cacheClient,
		docDBClient: docDBClient,
	}
}

// HealthResponse represents a health check response.
type HealthResponse struct {
	Status     string            `json:"status"`
	Components map[string]string `json:"components,omitempty"`
}

// Health handles the /health endpoint.
// @Summary Health check
// @Description Returns the overall health status and component statuses
// @Tags Health
// @Produce json
// @Success 200 {object} HealthResponse "Service healthy"
// @Failure 503 {object} HealthResponse "Service unhealthy"
// @Router /health [get]
func (h *HealthHandler) Health(c *gin.Context) {
	components := make(map[string]string)
	healthy := true

	// Check cache
	if err := h.cacheClient.Ping(c.Request.Context()); err != nil {
		components["cache"] = "unhealthy"
		healthy = false
	} else {
		components["cache"] = "healthy"
	}

	// Check document database
	if err := h.docDBClient.Ping(c.Request.Context()); err != nil {
		components["docdb"] = "unhealthy"
		healthy = false
	} else {
		components["docdb"] = "healthy"
	}

	status := "healthy"
	statusCode := http.StatusOK
	if !healthy {
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, HealthResponse{
		Status:     status,
		Components: components,
	})
}

// Ready handles the /ready endpoint.
// @Summary Readiness check
// @Description Returns 200 if the service is ready to accept traffic
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]string "Service ready"
// @Failure 503 {object} map[string]string "Service not ready"
// @Router /ready [get]
func (h *HealthHandler) Ready(c *gin.Context) {
	// Check all dependencies
	if err := h.cacheClient.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"reason": "cache unavailable",
		})
		return
	}

	if err := h.docDBClient.Ping(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"reason": "docdb unavailable",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}

// Live handles the /live endpoint.
// @Summary Liveness check
// @Description Returns 200 if the service is alive
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]string "Service alive"
// @Router /live [get]
func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
	})
}
