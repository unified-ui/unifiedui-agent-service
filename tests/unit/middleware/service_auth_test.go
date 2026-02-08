package middleware_test

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/unifiedui/agent-service/internal/api/middleware"
	"github.com/unifiedui/agent-service/internal/config"
	"github.com/unifiedui/agent-service/tests/mocks"
	"github.com/unifiedui/agent-service/tests/testutils"
)

func createServiceKeyMiddleware(mockVault *mocks.MockVaultClient) *middleware.ServiceKeyMiddleware {
	return middleware.NewServiceKeyMiddleware(mockVault, config.AppVaultConfig{
		PlatformServiceKey: "PLATFORM_TO_AGENT_SERVICE_KEY",
		AgentToPlatformKey: "AGENT_TO_PLATFORM_SERVICE_KEY",
	})
}

func TestServiceKeyMiddleware_ValidKey_Success(t *testing.T) {
	mockVault := mocks.NewMockVaultClient()
	mockVault.On("GetSecret", mock.Anything, "dotenv://PLATFORM_TO_AGENT_SERVICE_KEY", false).Return("valid-key", nil)

	mw := createServiceKeyMiddleware(mockVault)

	router := testutils.SetupTestRouter()
	router.Use(mw.AuthenticateServiceKey())
	router.DELETE("/test", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	headers := map[string]string{"X-Service-Key": "valid-key"}
	w := testutils.PerformRequest(router, "DELETE", "/test", nil, headers)

	testutils.AssertStatusCode(t, http.StatusNoContent, w)
	mockVault.AssertExpectations(t)
}

func TestServiceKeyMiddleware_InvalidKey_Forbidden(t *testing.T) {
	mockVault := mocks.NewMockVaultClient()
	mockVault.On("GetSecret", mock.Anything, "dotenv://PLATFORM_TO_AGENT_SERVICE_KEY", false).Return("valid-key", nil)

	mw := createServiceKeyMiddleware(mockVault)

	router := testutils.SetupTestRouter()
	router.Use(mw.AuthenticateServiceKey())
	router.DELETE("/test", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	headers := map[string]string{"X-Service-Key": "wrong-key"}
	w := testutils.PerformRequest(router, "DELETE", "/test", nil, headers)

	testutils.AssertStatusCode(t, http.StatusForbidden, w)
}

func TestServiceKeyMiddleware_MissingHeader_Unauthorized(t *testing.T) {
	mockVault := mocks.NewMockVaultClient()

	mw := createServiceKeyMiddleware(mockVault)

	router := testutils.SetupTestRouter()
	router.Use(mw.AuthenticateServiceKey())
	router.DELETE("/test", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	w := testutils.PerformRequest(router, "DELETE", "/test", nil, nil)

	testutils.AssertStatusCode(t, http.StatusUnauthorized, w)
}

func TestServiceKeyMiddleware_VaultError_InternalError(t *testing.T) {
	mockVault := mocks.NewMockVaultClient()
	mockVault.On("GetSecret", mock.Anything, "dotenv://PLATFORM_TO_AGENT_SERVICE_KEY", false).Return("", assert.AnError)

	mw := createServiceKeyMiddleware(mockVault)

	router := testutils.SetupTestRouter()
	router.Use(mw.AuthenticateServiceKey())
	router.DELETE("/test", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	headers := map[string]string{"X-Service-Key": "some-key"}
	w := testutils.PerformRequest(router, "DELETE", "/test", nil, headers)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)
}
