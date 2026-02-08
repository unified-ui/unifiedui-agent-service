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
		PlatformServiceKey: "platform-to-agent-service-key",
		AgentToPlatformKey: "agent-to-platform-service-key",
	})
}

func TestServiceKeyMiddleware_ValidKey_Success(t *testing.T) {
	mockVault := mocks.NewMockVaultClient()
	mockVault.On("GetSecret", mock.Anything, "dotenv://platform-to-agent-service-key", false).Return("valid-key", nil)

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
	mockVault.On("GetSecret", mock.Anything, "dotenv://platform-to-agent-service-key", false).Return("valid-key", nil)

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
	mockVault.On("GetSecret", mock.Anything, "dotenv://platform-to-agent-service-key", false).Return("", assert.AnError)

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
