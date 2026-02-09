package handlers

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/unifiedui/agent-service/internal/api/middleware"
	"github.com/unifiedui/agent-service/internal/core/docdb"
	"github.com/unifiedui/agent-service/internal/domain/errors"
	"github.com/unifiedui/agent-service/internal/domain/models"
)

func (h *TracesHandler) parseListTracesQueryParams(c *gin.Context) (*docdb.ListTracesOptions, *errors.DomainError) {
	limitStr := c.DefaultQuery("limit", "20")
	skipStr := c.DefaultQuery("skip", "0")
	order := c.DefaultQuery("order", "desc")
	orderByParam := c.DefaultQuery("order_by", "created_at")
	createdAfterStr := c.Query("created_after")
	createdBeforeStr := c.Query("created_before")
	expandStr := c.DefaultQuery("expand", "false")

	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil || limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	skip, err := strconv.ParseInt(skipStr, 10, 64)
	if err != nil || skip < 0 {
		skip = 0
	}

	sortOrder := docdb.SortOrderDesc
	if order == "asc" {
		sortOrder = docdb.SortOrderAsc
	}

	sortField := docdb.SortFieldCreatedAt
	if orderByParam == "updated_at" {
		sortField = docdb.SortFieldUpdatedAt
	}

	expand := expandStr == "true"

	opts := &docdb.ListTracesOptions{
		Limit:   limit,
		Skip:    skip,
		OrderBy: sortOrder,
		SortBy:  sortField,
		Expand:  expand,
	}

	if createdAfterStr != "" {
		t, err := time.Parse(time.RFC3339, createdAfterStr)
		if err != nil {
			return nil, errors.NewValidationError("invalid created_after format", "must be ISO 8601 / RFC 3339")
		}
		opts.CreatedAfter = &t
	}
	if createdBeforeStr != "" {
		t, err := time.Parse(time.RFC3339, createdBeforeStr)
		if err != nil {
			return nil, errors.NewValidationError("invalid created_before format", "must be ISO 8601 / RFC 3339")
		}
		opts.CreatedBefore = &t
	}

	return opts, nil
}

func (h *TracesHandler) getUserID(ctx context.Context, authToken string) (string, error) {
	if h.platformClient == nil {
		return "system", nil
	}

	userInfo, err := h.platformClient.GetMe(ctx, authToken)
	if err != nil {
		return "system", nil
	}

	return userInfo.ID, nil
}

func (h *TracesHandler) validateConversationContext(ctx context.Context, tenantID, _, conversationID, authToken string) *errors.DomainError {
	if h.platformClient == nil {
		return nil
	}

	if err := h.platformClient.ValidateConversation(ctx, tenantID, conversationID, authToken); err != nil {
		errStr := err.Error()
		if len(errStr) > 12 && errStr[:12] == "unauthorized" {
			return errors.NewUnauthorizedError("invalid or expired token")
		}
		if len(errStr) > 9 && errStr[:9] == "forbidden" {
			return errors.NewForbiddenError("access denied to conversation")
		}
		if len(errStr) > 9 && errStr[:9] == "not_found" {
			return errors.NewNotFoundError("conversation", conversationID)
		}
		return errors.NewInternalError("failed to validate conversation", err)
	}

	return nil
}

func (h *TracesHandler) resolveAutonomousAgentFromBearer(ctx context.Context, tenantID, agentID, authToken string) *errors.DomainError {
	if h.platformClient == nil {
		return nil
	}

	err := h.platformClient.ValidateAutonomousAgent(ctx, tenantID, agentID, authToken)
	if err != nil {
		errStr := err.Error()
		if strings.HasPrefix(errStr, "unauthorized") {
			return errors.NewUnauthorizedError("unauthorized access to autonomous agent")
		}
		if strings.HasPrefix(errStr, "forbidden") {
			return errors.NewForbiddenError("no permission to access autonomous agent")
		}
		if strings.HasPrefix(errStr, "not_found") {
			return errors.NewNotFoundError("autonomous agent", agentID)
		}
		return errors.NewInternalError("failed to validate autonomous agent access", err)
	}

	return nil
}

func (h *TracesHandler) resolveUserIDFromAPIKey(ctx context.Context, tenantID, agentID, apiKey string) *errors.DomainError {
	if h.platformClient == nil {
		return nil
	}

	err := h.platformClient.ValidateAutonomousAgentAPIKey(ctx, tenantID, agentID, apiKey)
	if err != nil {
		errStr := err.Error()
		if strings.HasPrefix(errStr, "unauthorized") {
			return errors.NewUnauthorizedError("invalid API key")
		}
		if strings.HasPrefix(errStr, "not_found") {
			return errors.NewNotFoundError("autonomous agent", agentID)
		}
		return errors.NewInternalError("failed to validate autonomous agent API key", err)
	}

	return nil
}

func (h *TracesHandler) resolveUserIDForTrace(ctx context.Context, c *gin.Context, tenantID string, trace *models.Trace) (string, *errors.DomainError) {
	authToken := middleware.GetToken(c)
	apiKey := middleware.GetAutonomousAgentAPIKey(c)

	if authToken != "" {
		uid, err := h.getUserID(ctx, authToken)
		if err != nil {
			return "", errors.NewInternalError("failed to get user info", err)
		}
		return uid, nil
	}

	if apiKey != "" {
		if trace.ContextType != models.TraceContextAutonomousAgent || trace.AutonomousAgentID == "" {
			return "", errors.NewForbiddenError("API key authentication is only allowed for autonomous agent traces")
		}

		if domainErr := h.resolveUserIDFromAPIKey(ctx, tenantID, trace.AutonomousAgentID, apiKey); domainErr != nil {
			return "", domainErr
		}
		return "autonomous-agent-" + trace.AutonomousAgentID, nil
	}

	return "", errors.NewUnauthorizedError("no valid authentication provided")
}
