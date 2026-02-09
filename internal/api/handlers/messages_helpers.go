package handlers

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/unifiedui/agent-service/internal/api/middleware"
	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/platform"
	"github.com/unifiedui/agent-service/internal/services/session"
	"github.com/unifiedui/agent-service/internal/services/traceimport"
)

func generateMessageID() string {
	return "msg_" + uuid.New().String()
}

func generateConversationID() string {
	return "conv_" + uuid.New().String()
}

func (h *MessagesHandler) saveFailedAssistantMessage(ctx context.Context, assistantMessage *models.Message, errorMsg string) {
	assistantMessage.SetError(errorMsg)
	_ = h.docDBClient.Messages().Add(ctx, assistantMessage)
}

func (h *MessagesHandler) saveAssistantMessageWithMetadata(
	ctx context.Context,
	msg *models.Message,
	agentConfig *platform.AgentConfig,
	startTime time.Time,
) *models.Message {
	latencyMs := time.Since(startTime).Milliseconds()

	if msg.Metadata == nil {
		msg.Metadata = &models.AssistantMetadata{}
	}
	msg.Metadata.LatencyMs = latencyMs
	msg.Metadata.AgentType = string(agentConfig.Type)

	if err := h.docDBClient.Messages().Add(ctx, msg); err != nil {
		return nil
	}
	return msg
}

func (h *MessagesHandler) extractFoundryMetadata(metadata map[string]interface{}) *models.AssistantMetadata {
	result := &models.AssistantMetadata{}

	if messageID, ok := metadata["message_id"].(string); ok {
		result.ExecutionID = messageID
		result.ExtMessageID = messageID
	}

	return result
}

func (h *MessagesHandler) mergeFoundryMetadata(msg *models.Message, metadata map[string]interface{}) {
	if msg.Metadata == nil {
		msg.Metadata = &models.AssistantMetadata{}
	}

	if responseID, ok := metadata["response_id"].(string); ok && msg.Metadata.ExecutionID == "" {
		msg.Metadata.ExecutionID = responseID
	}

	if messageID, ok := metadata["message_id"].(string); ok && msg.Metadata.ExtMessageID == "" {
		msg.Metadata.ExtMessageID = messageID
	}

	if model, ok := metadata["model"].(string); ok {
		msg.Metadata.Model = model
	}

	if agentName, ok := metadata["agent_name"].(string); ok {
		msg.Metadata.AgentType = agentName
	}

	if usage, ok := metadata["usage"].(map[string]interface{}); ok {
		if inputTokens, ok := usage["input_tokens"].(int); ok {
			msg.Metadata.TokensInput = inputTokens
		}
		if outputTokens, ok := usage["output_tokens"].(int); ok {
			msg.Metadata.TokensOutput = outputTokens
		}
	}

	if workflowType, ok := metadata["type"].(string); ok {
		if workflowType == "workflow_action" {
			msg.AddStatusTrace(
				"workflow_action",
				getStringFromMap(metadata, "kind"),
				"",
				map[string]interface{}{
					"action_id":          getStringFromMap(metadata, "action_id"),
					"parent_action_id":   getStringFromMap(metadata, "parent_action_id"),
					"previous_action_id": getStringFromMap(metadata, "previous_action_id"),
					"status":             getStringFromMap(metadata, "status"),
				},
			)
		}
	}
}

func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func (h *MessagesHandler) updateSessionCache(
	ctx context.Context,
	tenantCtx *middleware.TenantContext,
	agentConfig *platform.AgentConfig,
	userMessage *models.Message,
	assistantMessage *models.Message,
) {
	if !agentConfig.Settings.UseUnifiedChatHistory {
		return
	}

	sessionData, err := h.sessionService.GetSession(ctx, tenantCtx.TenantID, tenantCtx.UserID, userMessage.ConversationID)
	if err != nil || sessionData == nil {
		chatHistory := []models.ChatHistoryEntry{
			userMessage.ToChatHistoryEntry(),
			assistantMessage.ToChatHistoryEntry(),
		}
		sessionData = session.NewSessionData(
			agentConfig,
			chatHistory,
			tenantCtx.TenantID,
			tenantCtx.UserID,
			userMessage.ConversationID,
		)
		_ = h.sessionService.SetSession(ctx, sessionData)
		return
	}

	newEntries := []models.ChatHistoryEntry{
		userMessage.ToChatHistoryEntry(),
		assistantMessage.ToChatHistoryEntry(),
	}
	_ = h.sessionService.UpdateChatHistory(ctx, tenantCtx.TenantID, tenantCtx.UserID, userMessage.ConversationID, newEntries)
}

func (h *MessagesHandler) updateSessionCacheConfigOnly(
	ctx context.Context,
	tenantCtx *middleware.TenantContext,
	agentConfig *platform.AgentConfig,
	conversationID string,
) {
	existingSession, _ := h.sessionService.GetSession(ctx, tenantCtx.TenantID, tenantCtx.UserID, conversationID)
	if existingSession != nil {
		return
	}

	sessionData := session.NewSessionData(
		agentConfig,
		[]models.ChatHistoryEntry{},
		tenantCtx.TenantID,
		tenantCtx.UserID,
		conversationID,
	)
	_ = h.sessionService.SetSession(ctx, sessionData)
}

func (h *MessagesHandler) enqueueFoundryTraceImport(
	tenantCtx *middleware.TenantContext,
	agentConfig *platform.AgentConfig,
	userMessage *models.Message,
	extConversationID string,
	foundryAPIKey string,
) {
	req := traceimport.NewImportRequest(
		tenantCtx.TenantID,
		userMessage.ConversationID,
		userMessage.ApplicationID,
		tenantCtx.UserID,
	)

	req.WithBackendConfig("ext_conversation_id", extConversationID)
	req.WithBackendConfig("project_endpoint", agentConfig.Settings.ProjectEndpoint)
	req.WithBackendConfig("api_version", agentConfig.Settings.APIVersion)
	req.WithBackendConfig("api_token", foundryAPIKey)

	_ = h.importService.EnqueueImport(platform.AgentTypeFoundry, req)
}

func (h *MessagesHandler) enqueueN8NTraceImport(
	tenantCtx *middleware.TenantContext,
	agentConfig *platform.AgentConfig,
	userMessage *models.Message,
	executionID string,
) {
	apiKey := ""
	if agentConfig.Settings.APICredentials != nil {
		apiKey = agentConfig.Settings.APICredentials.GetSecretAsString()
	}

	if apiKey == "" {
		return
	}

	baseURL := extractBaseURL(agentConfig.Settings.ChatURL)
	if baseURL == "" {
		return
	}

	if executionID == "" && userMessage.ConversationID == "" {
		return
	}

	req := traceimport.NewImportRequest(
		tenantCtx.TenantID,
		userMessage.ConversationID,
		userMessage.ApplicationID,
		tenantCtx.UserID,
	)

	req.WithBackendConfig("execution_id", executionID)
	req.WithBackendConfig("session_id", userMessage.ConversationID)
	req.WithBackendConfig("base_url", baseURL)
	req.WithBackendConfig("api_key", apiKey)

	_ = h.importService.EnqueueImport(platform.AgentTypeN8N, req)
}

func extractBaseURL(fullURL string) string {
	if fullURL == "" {
		return ""
	}

	protocolEnd := 0
	if len(fullURL) > 8 && fullURL[:8] == "https://" {
		protocolEnd = 8
	} else if len(fullURL) > 7 && fullURL[:7] == "http://" {
		protocolEnd = 7
	}

	slashIdx := -1
	for i := protocolEnd; i < len(fullURL); i++ {
		if fullURL[i] == '/' {
			slashIdx = i
			break
		}
	}

	if slashIdx == -1 {
		return fullURL
	}

	return fullURL[:slashIdx]
}
