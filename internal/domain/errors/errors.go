// Package errors provides domain-specific error types.
package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Error codes for domain errors.
const (
	ErrCodeNotFound           = "NOT_FOUND"
	ErrCodeValidation         = "VALIDATION_ERROR"
	ErrCodeUnauthorized       = "UNAUTHORIZED"
	ErrCodeForbidden          = "FORBIDDEN"
	ErrCodeInternal           = "INTERNAL_ERROR"
	ErrCodeBadRequest         = "BAD_REQUEST"
	ErrCodeConflict           = "CONFLICT"
	ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	ErrCodeTimeout            = "TIMEOUT"
)

// DomainError represents a domain-specific error.
type DomainError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Details    string `json:"details,omitempty"`
	HTTPStatus int    `json:"-"`
	Err        error  `json:"-"`
}

// Error implements the error interface.
func (e *DomainError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error.
func (e *DomainError) Unwrap() error {
	return e.Err
}

// NewNotFoundError creates a new not found error.
func NewNotFoundError(resource, identifier string) *DomainError {
	return &DomainError{
		Code:       ErrCodeNotFound,
		Message:    fmt.Sprintf("%s not found", resource),
		Details:    identifier,
		HTTPStatus: http.StatusNotFound,
	}
}

// NewValidationError creates a new validation error.
func NewValidationError(message string, details string) *DomainError {
	return &DomainError{
		Code:       ErrCodeValidation,
		Message:    message,
		Details:    details,
		HTTPStatus: http.StatusBadRequest,
	}
}

// NewUnauthorizedError creates a new unauthorized error.
func NewUnauthorizedError(message string) *DomainError {
	return &DomainError{
		Code:       ErrCodeUnauthorized,
		Message:    message,
		HTTPStatus: http.StatusUnauthorized,
	}
}

// NewForbiddenError creates a new forbidden error.
func NewForbiddenError(message string) *DomainError {
	return &DomainError{
		Code:       ErrCodeForbidden,
		Message:    message,
		HTTPStatus: http.StatusForbidden,
	}
}

// NewInternalError creates a new internal error.
func NewInternalError(message string, err error) *DomainError {
	details := ""
	if err != nil {
		details = err.Error()
	}
	return &DomainError{
		Code:       ErrCodeInternal,
		Message:    message,
		Details:    details,
		HTTPStatus: http.StatusInternalServerError,
		Err:        err,
	}
}

// NewBadRequestError creates a new bad request error.
func NewBadRequestError(message string, details string) *DomainError {
	return &DomainError{
		Code:       ErrCodeBadRequest,
		Message:    message,
		Details:    details,
		HTTPStatus: http.StatusBadRequest,
	}
}

// NewConflictError creates a new conflict error.
func NewConflictError(message string, details string) *DomainError {
	return &DomainError{
		Code:       ErrCodeConflict,
		Message:    message,
		Details:    details,
		HTTPStatus: http.StatusConflict,
	}
}

// NewServiceUnavailableError creates a new service unavailable error.
func NewServiceUnavailableError(service string, err error) *DomainError {
	return &DomainError{
		Code:       ErrCodeServiceUnavailable,
		Message:    fmt.Sprintf("%s is unavailable", service),
		HTTPStatus: http.StatusServiceUnavailable,
		Err:        err,
	}
}

// NewTimeoutError creates a new timeout error.
func NewTimeoutError(operation string) *DomainError {
	return &DomainError{
		Code:       ErrCodeTimeout,
		Message:    fmt.Sprintf("%s timed out", operation),
		HTTPStatus: http.StatusGatewayTimeout,
	}
}

// IsDomainError checks if the error is a domain error.
func IsDomainError(err error) bool {
	var domainErr *DomainError
	return errors.As(err, &domainErr)
}

// GetDomainError extracts the domain error from an error.
func GetDomainError(err error) (*DomainError, bool) {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr, true
	}
	return nil, false
}

// IsNotFound checks if the error is a not found error.
func IsNotFound(err error) bool {
	domainErr, ok := GetDomainError(err)
	return ok && domainErr.Code == ErrCodeNotFound
}

// IsValidationError checks if the error is a validation error.
func IsValidationError(err error) bool {
	domainErr, ok := GetDomainError(err)
	return ok && domainErr.Code == ErrCodeValidation
}

// IsUnauthorized checks if the error is an unauthorized error.
func IsUnauthorized(err error) bool {
	domainErr, ok := GetDomainError(err)
	return ok && domainErr.Code == ErrCodeUnauthorized
}
