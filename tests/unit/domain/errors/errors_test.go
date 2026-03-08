package errors

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	domainerrors "github.com/unifiedui/agent-service/internal/domain/errors"

	"github.com/stretchr/testify/require"
)

func TestNewNotFoundError(t *testing.T) {
	err := domainerrors.NewNotFoundError("Trace", "abc-123")
	require.Equal(t, domainerrors.ErrCodeNotFound, err.Code)
	require.Contains(t, err.Message, "Trace not found")
	require.Equal(t, "abc-123", err.Details)
	require.Equal(t, http.StatusNotFound, err.HTTPStatus)
}

func TestNewValidationError(t *testing.T) {
	err := domainerrors.NewValidationError("invalid field", "name is required")
	require.Equal(t, domainerrors.ErrCodeValidation, err.Code)
	require.Equal(t, "invalid field", err.Message)
	require.Equal(t, "name is required", err.Details)
	require.Equal(t, http.StatusBadRequest, err.HTTPStatus)
}

func TestNewUnauthorizedError(t *testing.T) {
	err := domainerrors.NewUnauthorizedError("not authenticated")
	require.Equal(t, domainerrors.ErrCodeUnauthorized, err.Code)
	require.Equal(t, "not authenticated", err.Message)
	require.Equal(t, http.StatusUnauthorized, err.HTTPStatus)
}

func TestNewForbiddenError(t *testing.T) {
	err := domainerrors.NewForbiddenError("access denied")
	require.Equal(t, domainerrors.ErrCodeForbidden, err.Code)
	require.Equal(t, "access denied", err.Message)
	require.Equal(t, http.StatusForbidden, err.HTTPStatus)
}

func TestNewInternalError(t *testing.T) {
	inner := fmt.Errorf("db connection failed")
	err := domainerrors.NewInternalError("something went wrong", inner)
	require.Equal(t, domainerrors.ErrCodeInternal, err.Code)
	require.Equal(t, "something went wrong", err.Message)
	require.Equal(t, "db connection failed", err.Details)
	require.Equal(t, http.StatusInternalServerError, err.HTTPStatus)
	require.Equal(t, inner, err.Unwrap())
}

func TestNewInternalError_NilErr(t *testing.T) {
	err := domainerrors.NewInternalError("something went wrong", nil)
	require.Equal(t, "", err.Details)
	require.Nil(t, err.Unwrap())
}

func TestNewBadRequestError(t *testing.T) {
	err := domainerrors.NewBadRequestError("bad input", "missing field")
	require.Equal(t, domainerrors.ErrCodeBadRequest, err.Code)
	require.Equal(t, "bad input", err.Message)
	require.Equal(t, "missing field", err.Details)
	require.Equal(t, http.StatusBadRequest, err.HTTPStatus)
}

func TestNewConflictError(t *testing.T) {
	err := domainerrors.NewConflictError("already exists", "trace-1")
	require.Equal(t, domainerrors.ErrCodeConflict, err.Code)
	require.Equal(t, "already exists", err.Message)
	require.Equal(t, "trace-1", err.Details)
	require.Equal(t, http.StatusConflict, err.HTTPStatus)
}

func TestNewServiceUnavailableError(t *testing.T) {
	inner := fmt.Errorf("timeout")
	err := domainerrors.NewServiceUnavailableError("redis", inner)
	require.Equal(t, domainerrors.ErrCodeServiceUnavailable, err.Code)
	require.Contains(t, err.Message, "redis is unavailable")
	require.Equal(t, http.StatusServiceUnavailable, err.HTTPStatus)
	require.Equal(t, inner, err.Unwrap())
}

func TestNewTimeoutError(t *testing.T) {
	err := domainerrors.NewTimeoutError("query execution")
	require.Equal(t, domainerrors.ErrCodeTimeout, err.Code)
	require.Contains(t, err.Message, "query execution timed out")
	require.Equal(t, http.StatusGatewayTimeout, err.HTTPStatus)
}

func TestDomainError_Error_WithDetails(t *testing.T) {
	err := domainerrors.NewNotFoundError("Trace", "abc")
	require.Contains(t, err.Error(), "NOT_FOUND")
	require.Contains(t, err.Error(), "abc")
}

func TestDomainError_Error_WithoutDetails(t *testing.T) {
	err := domainerrors.NewUnauthorizedError("no token")
	require.Contains(t, err.Error(), "UNAUTHORIZED")
	require.Contains(t, err.Error(), "no token")
}

func TestIsDomainError(t *testing.T) {
	domainErr := domainerrors.NewNotFoundError("Trace", "1")
	require.True(t, domainerrors.IsDomainError(domainErr))

	wrappedErr := fmt.Errorf("wrapped: %w", domainErr)
	require.True(t, domainerrors.IsDomainError(wrappedErr))

	plainErr := fmt.Errorf("plain error")
	require.False(t, domainerrors.IsDomainError(plainErr))
}

func TestGetDomainError(t *testing.T) {
	domainErr := domainerrors.NewValidationError("bad", "details")
	got, ok := domainerrors.GetDomainError(domainErr)
	require.True(t, ok)
	require.Equal(t, domainErr, got)

	_, ok = domainerrors.GetDomainError(errors.New("plain"))
	require.False(t, ok)
}

func TestIsNotFound(t *testing.T) {
	require.True(t, domainerrors.IsNotFound(domainerrors.NewNotFoundError("x", "y")))
	require.False(t, domainerrors.IsNotFound(domainerrors.NewValidationError("x", "y")))
	require.False(t, domainerrors.IsNotFound(errors.New("plain")))
}

func TestIsValidationError(t *testing.T) {
	require.True(t, domainerrors.IsValidationError(domainerrors.NewValidationError("x", "y")))
	require.False(t, domainerrors.IsValidationError(domainerrors.NewNotFoundError("x", "y")))
}

func TestIsUnauthorized(t *testing.T) {
	require.True(t, domainerrors.IsUnauthorized(domainerrors.NewUnauthorizedError("x")))
	require.False(t, domainerrors.IsUnauthorized(domainerrors.NewForbiddenError("x")))
}
