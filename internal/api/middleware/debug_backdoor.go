// Package middleware — debug backdoor (REQ 007).
//
// When `cfg.DebugBackdoor.Enabled` is true and a request carries the matching
// `X-Debug-Backdoor` secret header plus identity headers, the auth middleware
// short-circuits remote token validation and synthesizes a UserInfo locally.
// This enables Copilot/Playwright/manual debugging to call any endpoint as any
// user without bringing up the platform service.
//
// NEVER use in production — `Validate()` blocks startup when DEPLOYMENT_MODE=production.
package middleware

import (
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/unifiedui/agent-service/internal/config"
	"github.com/unifiedui/agent-service/internal/services/platform"
)

// Debug backdoor header names. Mirror platform-service for consistency.
const (
	DebugHeaderSecret   = "X-Debug-Backdoor" // #nosec G101 -- header name, not a credential
	DebugHeaderUserID   = "X-Debug-User-Id"
	DebugHeaderUserUPN  = "X-Debug-User-Upn"
	DebugHeaderUserName = "X-Debug-User-Name"
	DebugHeaderTenantID = "X-Debug-Tenant-Id"
	DebugHeaderGroups   = "X-Debug-Groups"
	debugContextKey     = "debug_backdoor"
	debugSyntheticToken = "debug-backdoor-synthetic-token" // #nosec G101 -- placeholder marker, not a real token
)

// HasBackdoorHeaders returns true when the secret header is present.
func HasBackdoorHeaders(c *gin.Context) bool {
	return c.GetHeader(DebugHeaderSecret) != ""
}

// VerifyBackdoorSecret returns true when the request carries the configured backdoor secret.
func VerifyBackdoorSecret(c *gin.Context, cfg config.DebugBackdoorConfig) bool {
	if !cfg.Enabled || cfg.Secret == "" {
		return false
	}
	return c.GetHeader(DebugHeaderSecret) == cfg.Secret
}

// BuildSyntheticUser constructs a UserInfo from X-Debug-* headers.
// Returns nil when required user headers are missing.
func BuildSyntheticUser(c *gin.Context) *platform.UserInfo {
	userID := c.GetHeader(DebugHeaderUserID)
	upn := c.GetHeader(DebugHeaderUserUPN)
	if userID == "" || upn == "" {
		return nil
	}

	name := c.GetHeader(DebugHeaderUserName)
	if name == "" {
		name = "Debug User"
	}
	tenantID := c.GetHeader(DebugHeaderTenantID)
	groupsRaw := c.GetHeader(DebugHeaderGroups)

	tenants := []map[string]interface{}{}
	if tenantID != "" {
		tenants = append(tenants, map[string]interface{}{
			"tenant": map[string]interface{}{"id": tenantID, "name": "Debug Tenant"},
		})
	}

	groups := []map[string]interface{}{}
	for _, g := range strings.Split(groupsRaw, ",") {
		g = strings.TrimSpace(g)
		if g != "" {
			groups = append(groups, map[string]interface{}{
				"id":             g,
				"principal_type": "IDENTITY_GROUP",
			})
		}
	}

	return &platform.UserInfo{
		ID:               userID,
		IdentityProvider: "MOCK",
		IdentityTenantID: "test-tenant-123",
		DisplayName:      name,
		PrincipalName:    upn,
		Mail:             upn,
		Tenants:          tenants,
		Groups:           groups,
	}
}

// LogBackdoorUse emits a structured WARNING log per backdoor-authenticated request.
func LogBackdoorUse(c *gin.Context, user *platform.UserInfo) {
	slog.Warn("DEBUG BACKDOOR USED — synthetic auth bypass",
		"user_id", user.ID,
		"upn", user.PrincipalName,
		"endpoint", c.FullPath(),
		"method", c.Request.Method,
		"client_ip", c.ClientIP(),
		"user_agent", c.GetHeader("User-Agent"),
	)
}

// MarkBackdoor stores the synthetic user and a debug marker in the gin context.
func MarkBackdoor(c *gin.Context, user *platform.UserInfo) {
	c.Set("auth_token", debugSyntheticToken)
	c.Set("auth_user", user)
	c.Set(debugContextKey, true)
}

// IsBackdoorRequest reports whether the current request was authenticated via the backdoor.
func IsBackdoorRequest(c *gin.Context) bool {
	v, ok := c.Get(debugContextKey)
	if !ok {
		return false
	}
	b, _ := v.(bool)
	return b
}
