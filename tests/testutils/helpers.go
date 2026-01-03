// Package testutils provides test utilities and helpers.
package testutils

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// SetupTestRouter creates a new Gin router for testing.
func SetupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

// NewTestContext creates a new Gin context for testing.
func NewTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

// NewTestContextWithRequest creates a new Gin context with a request.
func NewTestContextWithRequest(method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()

	var req *http.Request
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	c, _ := gin.CreateTestContext(w)
	c.Request = req

	return c, w
}

// SetPathParams sets path parameters on a Gin context.
func SetPathParams(c *gin.Context, params map[string]string) {
	var ginParams []gin.Param
	for key, value := range params {
		ginParams = append(ginParams, gin.Param{Key: key, Value: value})
	}
	c.Params = ginParams
}

// SetAuthToken sets the auth token on a Gin context.
func SetAuthToken(c *gin.Context, token string) {
	c.Request.Header.Set("Authorization", "Bearer "+token)
	c.Set("auth_token", token)
}

// PerformRequest performs an HTTP request against a test router.
func PerformRequest(router *gin.Engine, method, path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	var req *http.Request
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// ParseJSONResponse parses a JSON response body.
func ParseJSONResponse(t *testing.T, w *httptest.ResponseRecorder, v interface{}) {
	t.Helper()
	err := json.Unmarshal(w.Body.Bytes(), v)
	require.NoError(t, err, "failed to parse JSON response")
}

// AssertStatusCode asserts the response status code.
func AssertStatusCode(t *testing.T, expected int, w *httptest.ResponseRecorder) {
	t.Helper()
	require.Equal(t, expected, w.Code, "unexpected status code: %s", w.Body.String())
}

// TestContext returns a context for testing with a timeout.
func TestContext() context.Context {
	return context.Background()
}

// RequireNoError is a helper to require no error.
func RequireNoError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	require.NoError(t, err, msgAndArgs...)
}

// RequireError is a helper to require an error.
func RequireError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	require.Error(t, err, msgAndArgs...)
}
