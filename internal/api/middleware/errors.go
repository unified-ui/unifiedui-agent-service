// Package middleware provides HTTP middleware for the API.
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	domainerrors "github.com/unifiedui/agent-service/internal/domain/errors"
)

// ErrorMiddleware handles error recovery and formatting.
type ErrorMiddleware struct{}

// NewErrorMiddleware creates a new ErrorMiddleware.
func NewErrorMiddleware() *ErrorMiddleware {
	return &ErrorMiddleware{}
}

// Recovery returns a gin middleware that recovers from panics.
func (m *ErrorMiddleware) Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Error().
					Interface("error", err).
					Str("path", c.Request.URL.Path).
					Str("method", c.Request.Method).
					Msg("panic recovered")

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code":    "INTERNAL_ERROR",
					"message": "internal server error",
				})
			}
		}()
		c.Next()
	}
}

// ErrorResponse represents a standardized error response.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// HandleError handles errors and sends appropriate HTTP responses.
func HandleError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	// Check for domain errors
	if domainErr, ok := domainerrors.GetDomainError(err); ok {
		c.AbortWithStatusJSON(domainErr.HTTPStatus, ErrorResponse{
			Code:    domainErr.Code,
			Message: domainErr.Message,
			Details: domainErr.Details,
		})
		return
	}

	// Default to internal server error
	log.Error().Err(err).Msg("unhandled error")
	c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{
		Code:    "INTERNAL_ERROR",
		Message: "internal server error",
	})
}

// NotFound returns a 404 handler.
func NotFound() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Code:    "NOT_FOUND",
			Message: "resource not found",
			Details: c.Request.URL.Path,
		})
	}
}

// MethodNotAllowed returns a 405 handler.
func MethodNotAllowed() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, ErrorResponse{
			Code:    "METHOD_NOT_ALLOWED",
			Message: "method not allowed",
			Details: c.Request.Method,
		})
	}
}
