// Package middleware provides HTTP middleware for the API.
package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// LoggingMiddleware handles request logging.
type LoggingMiddleware struct {
	logger zerolog.Logger
}

// NewLoggingMiddleware creates a new LoggingMiddleware.
func NewLoggingMiddleware() *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: log.Logger,
	}
}

// NewLoggingMiddlewareWithLogger creates a new LoggingMiddleware with a custom logger.
func NewLoggingMiddlewareWithLogger(logger zerolog.Logger) *LoggingMiddleware {
	return &LoggingMiddleware{
		logger: logger,
	}
}

// Logger returns a gin middleware that logs requests.
func (m *LoggingMiddleware) Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log after request
		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		// Build log event
		event := m.logger.Info()
		if status >= 400 && status < 500 {
			event = m.logger.Warn()
		} else if status >= 500 {
			event = m.logger.Error()
		}

		event.
			Str("method", method).
			Str("path", path).
			Str("query", query).
			Int("status", status).
			Dur("latency", latency).
			Str("client_ip", clientIP).
			Str("user_agent", c.Request.UserAgent()).
			Int("body_size", c.Writer.Size()).
			Msg("request completed")
	}
}

// RequestLogger logs detailed request information.
func (m *LoggingMiddleware) RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		// Set request ID in context
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// Create request-scoped logger
		requestLogger := m.logger.With().
			Str("request_id", requestID).
			Str("tenant_id", c.Param("tenantId")).
			Logger()

		c.Set("logger", requestLogger)

		c.Next()
	}
}

// GetRequestLogger retrieves the request-scoped logger from context.
func GetRequestLogger(c *gin.Context) zerolog.Logger {
	if logger, exists := c.Get("logger"); exists {
		return logger.(zerolog.Logger)
	}
	return log.Logger
}

// GetRequestID retrieves the request ID from context.
func GetRequestID(c *gin.Context) string {
	if requestID, exists := c.Get("request_id"); exists {
		return requestID.(string)
	}
	return ""
}

// generateRequestID generates a simple request ID.
func generateRequestID() string {
	return time.Now().Format("20060102150405.000000")
}
