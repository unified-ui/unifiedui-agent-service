package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/unifiedui/agent-service/internal/config"
	"github.com/unifiedui/agent-service/internal/core/vault"
)

// ServiceKeyMiddleware handles X-Service-Key authentication for internal service-to-service calls.
type ServiceKeyMiddleware struct {
	vaultClient vault.Client
	appVaultCfg config.AppVaultConfig
}

// NewServiceKeyMiddleware creates a new ServiceKeyMiddleware.
func NewServiceKeyMiddleware(vaultClient vault.Client, appVaultCfg config.AppVaultConfig) *ServiceKeyMiddleware {
	return &ServiceKeyMiddleware{
		vaultClient: vaultClient,
		appVaultCfg: appVaultCfg,
	}
}

// AuthenticateServiceKey returns a gin middleware that validates the X-Service-Key header
// against the expected service key stored in the vault.
func (m *ServiceKeyMiddleware) AuthenticateServiceKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceKey := c.GetHeader("X-Service-Key")
		if serviceKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHORIZED",
				"message": "missing X-Service-Key header",
			})
			return
		}

		expectedKey, err := m.resolveExpectedKey(c.Request.Context())
		if err != nil || expectedKey == "" {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "service key validation not configured",
			})
			return
		}

		if serviceKey != expectedKey {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    "FORBIDDEN",
				"message": "invalid service key",
			})
			return
		}

		c.Next()
	}
}

func (m *ServiceKeyMiddleware) resolveExpectedKey(ctx context.Context) (string, error) {
	keyName := m.appVaultCfg.PlatformServiceKey
	if keyName != "" && m.vaultClient != nil {
		uri := m.vaultClient.BuildSecretURI(keyName)
		secret, err := m.vaultClient.GetSecret(ctx, uri, false)
		if err == nil && secret != "" {
			return secret, nil
		}
	}
	return "", nil
}
