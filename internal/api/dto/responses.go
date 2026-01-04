// Package dto provides Data Transfer Objects for API requests and responses.
package dto

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// HealthResponse represents a health check response.
type HealthResponse struct {
	Status     string            `json:"status"`
	Components map[string]string `json:"components,omitempty"`
}

// PaginatedResponse represents a paginated response.
type PaginatedResponse struct {
	Data   interface{} `json:"data"`
	Total  int64       `json:"total"`
	Limit  int64       `json:"limit"`
	Offset int64       `json:"offset"`
}

// GetMessagesResponse represents the response for getting messages.
type GetMessagesResponse struct {
	Messages []*MessageResponse `json:"messages"`
	Total    int64              `json:"total"`
	Limit    int64              `json:"limit"`
	Offset   int64              `json:"offset"`
}

// SendMessageResponse represents the response for sending a message.
type SendMessageResponse struct {
	Message *MessageResponse `json:"message"`
}

// GetTracesResponse represents the response for getting traces.
type GetTracesResponse struct {
	Traces []*TraceResponse `json:"traces"`
	Total  int64            `json:"total"`
}

// UpdateTracesResponse represents the response for updating traces.
type UpdateTracesResponse struct {
	Updated int `json:"updated"`
	Created int `json:"created"`
}

// SSEMessageEvent represents an SSE message event.
type SSEMessageEvent struct {
	Content   string `json:"content"`
	MessageID string `json:"messageId,omitempty"`
	Done      bool   `json:"done"`
}

// SSETraceEvent represents an SSE trace event.
type SSETraceEvent struct {
	TraceID string      `json:"traceId"`
	Type    string      `json:"type"`
	Name    string      `json:"name"`
	Status  string      `json:"status"`
	Data    interface{} `json:"data,omitempty"`
}

// SSEErrorEvent represents an SSE error event.
type SSEErrorEvent struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}
