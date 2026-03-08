// Package middleware provides HTTP middleware for the API.
package middleware

import (
	"regexp"

	"github.com/gin-gonic/gin"
)

var validPathParam = regexp.MustCompile(`^[A-Za-z0-9_\-]+$`)

// SanitizePathParam validates and returns a path parameter value.
func SanitizePathParam(c *gin.Context, name string) string {
	val := c.Param(name)
	if val == "" {
		return ""
	}
	if !validPathParam.MatchString(val) {
		return ""
	}
	return val
}

// TenantMiddleware extracts tenant context from the request.
type TenantMiddleware struct{}

// NewTenantMiddleware creates a new TenantMiddleware.
func NewTenantMiddleware() *TenantMiddleware {
	return &TenantMiddleware{}
}

// ExtractTenant returns a gin middleware that extracts tenant ID from the path.
func (m *TenantMiddleware) ExtractTenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := SanitizePathParam(c, "tenantId")
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
	return SanitizePathParam(c, "tenantId")
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
		TenantID:       SanitizePathParam(c, "tenantId"),
		ConversationID: SanitizePathParam(c, "conversationId"),
		MessageID:      SanitizePathParam(c, "messageId"),
		AgentID:        SanitizePathParam(c, "agentId"),
		UserID:         c.GetString("user_id"),
	}
}
