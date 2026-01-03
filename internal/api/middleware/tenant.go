// Package middleware provides HTTP middleware for the API.
package middleware

import (
	"github.com/gin-gonic/gin"
)

// TenantMiddleware extracts tenant context from the request.
type TenantMiddleware struct{}

// NewTenantMiddleware creates a new TenantMiddleware.
func NewTenantMiddleware() *TenantMiddleware {
	return &TenantMiddleware{}
}

// ExtractTenant returns a gin middleware that extracts tenant ID from the path.
func (m *TenantMiddleware) ExtractTenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.Param("tenantId")
		if tenantID != "" {
			c.Set("tenant_id", tenantID)
		}
		c.Next()
	}
}

// GetTenantID retrieves the tenant ID from the gin context.
func GetTenantID(c *gin.Context) string {
	if tenantID, exists := c.Get("tenant_id"); exists {
		return tenantID.(string)
	}
	return c.Param("tenantId")
}

// TenantContext holds tenant-related context.
type TenantContext struct {
	TenantID       string
	ConversationID string
	MessageID      string
	AgentID        string
	UserID         string
}

// GetTenantContext extracts the full tenant context from the request.
func GetTenantContext(c *gin.Context) *TenantContext {
	return &TenantContext{
		TenantID:       c.Param("tenantId"),
		ConversationID: c.Param("conversationId"),
		MessageID:      c.Param("messageId"),
		AgentID:        c.Param("agentId"),
		UserID:         c.GetString("user_id"),
	}
}
