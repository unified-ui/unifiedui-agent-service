package handlers

import (
	"context"
	stderrors "errors"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/unifiedui/agent-service/internal/api/middleware"
	"github.com/unifiedui/agent-service/internal/api/sse"
	"github.com/unifiedui/agent-service/internal/core/docdb"
	"github.com/unifiedui/agent-service/internal/domain/errors"
	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/agents"
	"github.com/unifiedui/agent-service/internal/services/platform"
)

// SendMessage handles POST /tenants/{tenantId}/conversation/messages
// @Summary Send a message
// @Description Sends a message to an AI agent and returns the response via SSE streaming
// @Tags Messages
// @Accept json
// @Produce text/event-stream
// @Param tenantId path string true "Tenant ID"
// @Param X-Microsoft-Foundry-API-Key header string false "Microsoft Foundry API Key (required for Foundry agents)"
// @Param request body SendMessageRequest true "Message content with chatAgentId"
// @Success 200 {object} SendMessageResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /api/v1/agent-service/tenants/{tenantId}/conversation/messages [post]
func (h *MessagesHandler) SendMessage(c *gin.Context) {
	ctx := c.Request.Context()
	tenantCtx := middleware.GetTenantContext(c)

	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.HandleError(c, errors.NewValidationError("invalid request body", err.Error()))
		return
	}

	conversationID := req.ConversationID
	if conversationID == "" {
		conversationID = generateConversationID()
	}

	userMessageID := generateMessageID()
	assistantMessageID := generateMessageID()

	sessionData, err := h.sessionService.GetSession(ctx, tenantCtx.TenantID, tenantCtx.UserID, conversationID)
	if err != nil {
		sessionData = nil
	}

	useCache := c.GetHeader("X-Use-Cache") != "false"

	var agentConfig *platform.AgentConfig
	var chatHistory []models.ChatHistoryEntry

	if sessionData != nil {
		agentConfig = sessionData.Config
		chatHistory = sessionData.ChatHistory
	} else {
		authToken := middleware.GetToken(c)
		if authToken == "" {
			middleware.HandleError(c, errors.NewUnauthorizedError("auth token not found in context"))
			return
		}

		agentConfig, err = h.platformClient.GetAgentConfig(ctx, tenantCtx.TenantID, req.ChatAgentID, conversationID, authToken, useCache)
		if err != nil {
			middleware.HandleError(c, errors.NewInternalError("failed to get agent configuration", err))
			return
		}

		if useCache {
			_ = h.configCache.Set(ctx, tenantCtx.TenantID, tenantCtx.UserID, req.ChatAgentID, agentConfig)
		}

		if agentConfig.Settings.UseUnifiedChatHistory && agentConfig.Type != platform.AgentTypeFoundry {
			chatHistoryCount := agentConfig.Settings.ChatHistoryCount
			if chatHistoryCount == 0 {
				chatHistoryCount = DefaultChatHistoryCount
			}

			listOpts := &docdb.ListMessagesOptions{
				ConversationID: conversationID,
				TenantID:       tenantCtx.TenantID,
				Limit:          int64(chatHistoryCount),
				OrderBy:        docdb.SortOrderAsc,
			}

			chatHistory, err = h.docDBClient.Messages().ListChatHistory(ctx, listOpts)
			if err != nil {
				chatHistory = []models.ChatHistoryEntry{}
			}
		}
	}

	userMessage := models.NewUserMessage(
		tenantCtx.TenantID,
		conversationID,
		req.ChatAgentID,
		tenantCtx.UserID,
		req.Message.Content,
		req.Message.Attachments,
		&models.MessageRequest{
			ChatAgentID:    req.ChatAgentID,
			ConversationID: req.ConversationID,
			Message: models.MessageRequestContent{
				Content:     req.Message.Content,
				Attachments: req.Message.Attachments,
			},
			InvokeConfig: models.MessageInvokeConfig{
				ChatHistoryMessageCount: req.InvokeConfig.ChatHistoryMessageCount,
			},
			Extra: req.Extra,
		},
	)
	userMessage.ID = userMessageID
	userMessage.AttachmentsMetadata = convertFilesToAttachmentMetadata(req.Message.Files)

	if err := h.docDBClient.Messages().Add(ctx, userMessage); err != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to store user message", err))
		return
	}

	var agentClients *agents.AgentClients
	var createErr error

	if agentConfig.Type == platform.AgentTypeFoundry {
		foundryAPIKey := c.GetHeader("X-Microsoft-Foundry-API-Key")
		if foundryAPIKey == "" {
			middleware.HandleError(c, errors.NewValidationError("X-Microsoft-Foundry-API-Key header is required for Foundry agents", ""))
			return
		}
		agentClients, createErr = h.agentFactory.CreateFoundryClients(agentConfig, foundryAPIKey)
	} else if agentConfig.Type == platform.AgentTypeRestAPI {
		authToken := middleware.GetToken(c)
		agentClients, createErr = h.agentFactory.CreateRestAPIClients(agentConfig, authToken)
	} else {
		agentClients, createErr = h.agentFactory.CreateClients(agentConfig)
	}

	if createErr != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to create agent clients", createErr))
		return
	}
	defer func() { _ = agentClients.Close() }()

	assistantMessage := models.NewAssistantMessage(
		tenantCtx.TenantID,
		conversationID,
		userMessageID,
		req.ChatAgentID,
		"",
		models.MessageStatusPending,
	)
	assistantMessage.ID = assistantMessageID

	foundryAPIKey := c.GetHeader("X-Microsoft-Foundry-API-Key")
	authToken := middleware.GetToken(c)

	extConversationID := req.ExtConversationID
	if agentConfig.Type == platform.AgentTypeRestAPI && extConversationID == "" {
		if creator, ok := agentClients.WorkflowClient.(agents.ConversationCreator); ok {
			extID, convErr := creator.CreateConversation(ctx)
			if convErr != nil {
				log.Warn().Err(convErr).Msg("failed to create external conversation, proceeding without it")
			} else {
				extConversationID = extID
			}
		}
	}

	isFirstMessage := len(chatHistory) == 0
	if agentConfig.Type == platform.AgentTypeFoundry {
		count, err := h.docDBClient.Messages().CountByConversation(ctx, tenantCtx.TenantID, conversationID)
		if err == nil {
			isFirstMessage = count == 1
		}
	}
	files := convertFilesToFileInputs(req.Message.Files)
	h.handleStreamingResponse(c, tenantCtx, agentClients, agentConfig, userMessage, assistantMessage, chatHistory, extConversationID, foundryAPIKey, req.InvokeConfig.ContextData, authToken, isFirstMessage, files)
}

func (h *MessagesHandler) handleStreamingResponse(
	c *gin.Context,
	tenantCtx *middleware.TenantContext,
	agentClients *agents.AgentClients,
	agentConfig *platform.AgentConfig,
	userMessage *models.Message,
	assistantMessage *models.Message,
	chatHistory []models.ChatHistoryEntry,
	extConversationID string,
	foundryAPIKey string,
	contextData map[string]string,
	authToken string,
	isFirstMessage bool,
	files []agents.FileInput,
) {
	ctx := c.Request.Context()

	writer, err := sse.NewWriter(c.Writer)
	if err != nil {
		middleware.HandleError(c, errors.NewInternalError("streaming not supported", err))
		return
	}

	conversationIDForInvoke := userMessage.ConversationID
	if agentConfig.Type == platform.AgentTypeFoundry || agentConfig.Type == platform.AgentTypeRestAPI {
		conversationIDForInvoke = extConversationID
	}

	invokeReq := &agents.InvokeRequest{
		ConversationID: conversationIDForInvoke,
		Message:        userMessage.Content,
		SessionID:      userMessage.ConversationID,
		ChatHistory:    chatHistory,
		ContextData:    contextData,
		Files:          files,
	}

	streamReader, err := agentClients.WorkflowClient.InvokeStreamReader(ctx, invokeReq)
	if err != nil {
		errorMsg := "Failed to invoke agent: " + err.Error()
		_ = writer.WriteStreamStart(assistantMessage.ID, userMessage.ConversationID)
		_ = writer.WriteStreamError("STREAM_ERROR", errorMsg, err.Error())
		_ = writer.WriteStreamEnd()
		h.saveFailedAssistantMessage(ctx, assistantMessage, errorMsg)
		_ = writer.WriteMessageComplete(assistantMessage)
		return
	}
	defer func() { _ = streamReader.Close() }()

	startTime := time.Now()

	switch agentConfig.Type {
	case platform.AgentTypeFoundry:
		h.handleFoundryStreaming(ctx, writer, streamReader, tenantCtx, agentConfig, userMessage, assistantMessage, startTime)
		h.updateSessionCacheConfigOnly(ctx, tenantCtx, agentConfig, userMessage.ConversationID)

		if h.importService != nil && extConversationID != "" && foundryAPIKey != "" {
			h.enqueueFoundryTraceImport(tenantCtx, agentConfig, userMessage, extConversationID, foundryAPIKey)
		}
	case platform.AgentTypeReactAgent, platform.AgentTypeRestAPI:
		h.handleReActStreaming(ctx, writer, streamReader, tenantCtx, agentConfig, userMessage, assistantMessage, startTime)
		h.updateSessionCache(ctx, tenantCtx, agentConfig, userMessage, assistantMessage)
	default:
		executionID := h.handleDefaultStreaming(ctx, writer, streamReader, tenantCtx, agentConfig, userMessage, assistantMessage, startTime)
		h.updateSessionCache(ctx, tenantCtx, agentConfig, userMessage, assistantMessage)

		if h.importService != nil && agentConfig.Type == platform.AgentTypeN8N {
			h.enqueueN8NTraceImport(tenantCtx, agentConfig, userMessage, executionID)
		}
	}

	if isFirstMessage && h.aiService != nil {
		h.streamTitleGeneration(ctx, writer, tenantCtx.TenantID, userMessage.ConversationID, userMessage.Content, assistantMessage.Content, authToken)
	}
}

func (h *MessagesHandler) handleDefaultStreaming(
	ctx context.Context,
	writer *sse.Writer,
	streamReader agents.StreamReader,
	_ *middleware.TenantContext,
	agentConfig *platform.AgentConfig,
	userMessage *models.Message,
	assistantMessage *models.Message,
	startTime time.Time,
) string {
	var fullContent string
	var executionID string

	_ = writer.WriteStreamStart(assistantMessage.ID, userMessage.ConversationID)

	for {
		select {
		case <-ctx.Done():
			_ = streamReader.Close()
			_ = writer.WriteStreamEnd()
			h.saveCanceledAssistantMessage(assistantMessage, fullContent, agentConfig, startTime)
			return executionID
		default:
		}

		chunk, err := streamReader.Read()
		if stderrors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			if ctx.Err() != nil {
				_ = streamReader.Close()
				h.saveCanceledAssistantMessage(assistantMessage, fullContent, agentConfig, startTime)
				return executionID
			}
			errorMsg := "Stream error: " + err.Error()
			_ = writer.WriteStreamError("STREAM_ERROR", errorMsg, err.Error())
			h.saveFailedAssistantMessage(ctx, assistantMessage, errorMsg)
			_ = writer.WriteStreamEnd()
			_ = writer.WriteMessageComplete(assistantMessage)
			return executionID
		}

		switch chunk.Type {
		case agents.ChunkTypeContent:
			fullContent += chunk.Content
			_ = writer.WriteTextStream(chunk.Content)
		case agents.ChunkTypeMetadata:
			if chunk.ExecutionID != "" {
				executionID = chunk.ExecutionID
				if assistantMessage.Metadata == nil {
					assistantMessage.Metadata = &models.AssistantMetadata{}
				}
				assistantMessage.Metadata.ExecutionID = chunk.ExecutionID
			}
		case agents.ChunkTypeError:
			if chunk.Error != nil {
				errorMsg := chunk.Error.Error()
				_ = writer.WriteStreamError("CHUNK_ERROR", errorMsg, errorMsg)
				_ = writer.WriteStreamEnd()
				h.saveFailedAssistantMessage(ctx, assistantMessage, errorMsg)
				_ = writer.WriteMessageComplete(assistantMessage)
				return executionID
			}
		}
	}

	_ = writer.WriteStreamEnd()

	latencyMs := time.Since(startTime).Milliseconds()

	assistantMessage.SetSuccess(fullContent)
	if assistantMessage.Metadata == nil {
		assistantMessage.Metadata = &models.AssistantMetadata{}
	}
	assistantMessage.Metadata.LatencyMs = latencyMs
	assistantMessage.Metadata.AgentType = string(agentConfig.Type)

	if err := h.docDBClient.Messages().Add(ctx, assistantMessage); err == nil {
		_ = writer.WriteMessageComplete(assistantMessage)
	}

	return executionID
}

func (h *MessagesHandler) handleFoundryStreaming(
	ctx context.Context,
	writer *sse.Writer,
	streamReader agents.StreamReader,
	tenantCtx *middleware.TenantContext,
	agentConfig *platform.AgentConfig,
	userMessage *models.Message,
	currentMessage *models.Message,
	startTime time.Time,
) {
	var currentContent string
	var traces []models.StatusTrace
	firstMessage := currentMessage
	allMessages := []*models.Message{currentMessage}

	saveCurrentAndStartNew := func() {
		if currentContent != "" {
			currentMessage.SetSuccess(currentContent)
			h.saveAssistantMessageWithMetadata(ctx, currentMessage, agentConfig, startTime)
		}

		_ = writer.WriteJSON(sse.EventMessage, &sse.StreamMessage{
			Type: "STREAM_NEW_MESSAGE",
		})

		currentMessage = models.NewAssistantMessage(
			tenantCtx.TenantID,
			userMessage.ConversationID,
			userMessage.ID,
			userMessage.ChatAgentID,
			"",
			models.MessageStatusPending,
		)
		currentMessage.ID = generateMessageID()
		allMessages = append(allMessages, currentMessage)
		currentContent = ""

		_ = writer.WriteStreamStart(currentMessage.ID, userMessage.ConversationID)
	}

	_ = writer.WriteStreamStart(currentMessage.ID, userMessage.ConversationID)

	for {
		select {
		case <-ctx.Done():
			_ = streamReader.Close()
			_ = writer.WriteStreamEnd()
			firstMessage.StatusTraces = traces
			h.saveCanceledAssistantMessage(currentMessage, currentContent, agentConfig, startTime)
			return
		default:
		}

		chunk, err := streamReader.Read()
		if stderrors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			if ctx.Err() != nil {
				_ = streamReader.Close()
				firstMessage.StatusTraces = traces
				h.saveCanceledAssistantMessage(currentMessage, currentContent, agentConfig, startTime)
				return
			}
			errorMsg := "Stream error: " + err.Error()
			_ = writer.WriteStreamError("STREAM_ERROR", errorMsg, err.Error())
			firstMessage.StatusTraces = traces
			h.saveFailedAssistantMessage(ctx, currentMessage, errorMsg)
			_ = writer.WriteStreamEnd()
			_ = writer.WriteMessageComplete(currentMessage)
			return
		}

		switch chunk.Type {
		case agents.ChunkTypeContent:
			currentContent += chunk.Content
			_ = writer.WriteTextStream(chunk.Content)

		case agents.ChunkTypeNewMessage:
			firstMessage.StatusTraces = traces
			saveCurrentAndStartNew()
			if chunk.Metadata != nil {
				currentMessage.Metadata = h.extractFoundryMetadata(chunk.Metadata)
			}

		case agents.ChunkTypeToolCallStart:
			toolName := extractConfigString(chunk.Config, "tool_name")
			traces = appendStatusTrace(traces, "tool_call_start", toolName, chunk.Config)
			_ = writer.WriteToolCallStart(toolName, chunk.Config)
		case agents.ChunkTypeToolCallStream:
			traces = appendToLastTrace(traces, chunk.Content)
			_ = writer.WriteToolCallStream(chunk.Content)
		case agents.ChunkTypeToolCallEnd:
			traces = appendStatusTrace(traces, "tool_call_end", extractConfigString(chunk.Config, "tool_name"), chunk.Config)
			_ = writer.WriteToolCallEnd(chunk.Config)

		case agents.ChunkTypeSubAgentStart:
			agentName := extractConfigString(chunk.Config, "agent_name")
			traces = appendStatusTrace(traces, "sub_agent_start", agentName, chunk.Config)
			_ = writer.WriteSubAgentStart(agentName, chunk.Config)
		case agents.ChunkTypeSubAgentEnd:
			traces = appendStatusTrace(traces, "sub_agent_end", "", chunk.Config)
			_ = writer.WriteSubAgentEnd(chunk.Config)

		case agents.ChunkTypeMetadata:
			if currentMessage.Metadata == nil {
				currentMessage.Metadata = &models.AssistantMetadata{}
			}
			if chunk.ExecutionID != "" {
				currentMessage.Metadata.ExecutionID = chunk.ExecutionID
			}
			if chunk.Metadata != nil {
				h.mergeFoundryMetadata(currentMessage, chunk.Metadata)
			}

		case agents.ChunkTypeDone:
			if chunk.Metadata != nil {
				h.mergeFoundryMetadata(currentMessage, chunk.Metadata)
			}
			if chunk.ExecutionID != "" && currentMessage.Metadata != nil {
				currentMessage.Metadata.ExecutionID = chunk.ExecutionID
			}

		case agents.ChunkTypeError:
			if chunk.Error != nil {
				errorMsg := chunk.Error.Error()
				_ = writer.WriteStreamError("CHUNK_ERROR", errorMsg, errorMsg)
				_ = writer.WriteStreamEnd()
				firstMessage.StatusTraces = traces
				h.saveFailedAssistantMessage(ctx, currentMessage, errorMsg)
				_ = writer.WriteMessageComplete(currentMessage)
				return
			}
		}
	}

	_ = writer.WriteStreamEnd()

	firstMessage.StatusTraces = traces
	currentMessage.SetSuccess(currentContent)
	savedMsg := h.saveAssistantMessageWithMetadata(ctx, currentMessage, agentConfig, startTime)
	if savedMsg != nil {
		_ = writer.WriteMessageComplete(savedMsg)
	}
}

func (h *MessagesHandler) handleReActStreaming(
	ctx context.Context,
	writer *sse.Writer,
	streamReader agents.StreamReader,
	_ *middleware.TenantContext,
	agentConfig *platform.AgentConfig,
	userMessage *models.Message,
	assistantMessage *models.Message,
	startTime time.Time,
) {
	var fullContent string
	var traces []models.StatusTrace

	_ = writer.WriteStreamStart(assistantMessage.ID, userMessage.ConversationID)

	for {
		select {
		case <-ctx.Done():
			_ = streamReader.Close()
			_ = writer.WriteStreamEnd()
			assistantMessage.StatusTraces = traces
			h.saveCanceledAssistantMessage(assistantMessage, fullContent, agentConfig, startTime)
			return
		default:
		}

		chunk, err := streamReader.Read()
		if stderrors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			if ctx.Err() != nil {
				_ = streamReader.Close()
				assistantMessage.StatusTraces = traces
				h.saveCanceledAssistantMessage(assistantMessage, fullContent, agentConfig, startTime)
				return
			}
			errorMsg := "Stream error: " + err.Error()
			_ = writer.WriteStreamError("STREAM_ERROR", errorMsg, err.Error())
			assistantMessage.StatusTraces = traces
			h.saveFailedAssistantMessage(ctx, assistantMessage, errorMsg)
			_ = writer.WriteStreamEnd()
			_ = writer.WriteMessageComplete(assistantMessage)
			return
		}

		switch chunk.Type {
		case agents.ChunkTypeContent:
			fullContent += chunk.Content
			_ = writer.WriteTextStream(chunk.Content)

		case agents.ChunkTypeReasoningStart:
			traces = appendStatusTrace(traces, "reasoning_start", "", chunk.Config)
			_ = writer.WriteReasoningStart()
		case agents.ChunkTypeReasoningStream:
			traces = appendToLastTrace(traces, chunk.Content)
			_ = writer.WriteReasoningStream(chunk.Content)
		case agents.ChunkTypeReasoningEnd:
			traces = appendStatusTrace(traces, "reasoning_end", "", nil)
			_ = writer.WriteReasoningEnd()

		case agents.ChunkTypeToolCallStart:
			toolName := extractConfigString(chunk.Config, "tool_name")
			traces = appendStatusTrace(traces, "tool_call_start", toolName, chunk.Config)
			_ = writer.WriteToolCallStart(toolName, chunk.Config)
		case agents.ChunkTypeToolCallStream:
			traces = appendToLastTrace(traces, chunk.Content)
			_ = writer.WriteToolCallStream(chunk.Content)
		case agents.ChunkTypeToolCallEnd:
			traces = appendStatusTrace(traces, "tool_call_end", extractConfigString(chunk.Config, "tool_name"), chunk.Config)
			_ = writer.WriteToolCallEnd(chunk.Config)

		case agents.ChunkTypePlanStart:
			traces = appendStatusTrace(traces, "plan_start", "", nil)
			_ = writer.WritePlanStart()
		case agents.ChunkTypePlanStream:
			traces = appendToLastTrace(traces, chunk.Content)
			_ = writer.WritePlanStream(chunk.Content)
		case agents.ChunkTypePlanComplete:
			traces = appendStatusTrace(traces, "plan_end", "", chunk.Config)
			_ = writer.WritePlanComplete(chunk.Config)

		case agents.ChunkTypeSubAgentStart:
			agentName := extractConfigString(chunk.Config, "agent_name")
			traces = appendStatusTrace(traces, "sub_agent_start", agentName, chunk.Config)
			_ = writer.WriteSubAgentStart(agentName, chunk.Config)
		case agents.ChunkTypeSubAgentStream:
			traces = appendToLastTrace(traces, chunk.Content)
			_ = writer.WriteSubAgentStream(chunk.Content)
		case agents.ChunkTypeSubAgentEnd:
			traces = appendStatusTrace(traces, "sub_agent_end", "", chunk.Config)
			_ = writer.WriteSubAgentEnd(chunk.Config)

		case agents.ChunkTypeSynthesisStart:
			traces = appendStatusTrace(traces, "synthesis_start", "", nil)
			_ = writer.WriteSynthesisStart()
		case agents.ChunkTypeSynthesisStream:
			fullContent += chunk.Content
			traces = appendToLastTrace(traces, chunk.Content)
			_ = writer.WriteSynthesisStream(chunk.Content)

		case agents.ChunkTypeTrace:
			_ = writer.WriteStreamTrace(chunk.Config)

		case agents.ChunkTypeError:
			if chunk.Error != nil {
				errorMsg := chunk.Error.Error()
				_ = writer.WriteStreamError("CHUNK_ERROR", errorMsg, errorMsg)
				_ = writer.WriteStreamEnd()
				assistantMessage.StatusTraces = traces
				h.saveFailedAssistantMessage(ctx, assistantMessage, errorMsg)
				_ = writer.WriteMessageComplete(assistantMessage)
				return
			}
		}
	}

	_ = writer.WriteStreamEnd()

	latencyMs := time.Since(startTime).Milliseconds()

	assistantMessage.StatusTraces = traces
	assistantMessage.SetSuccess(fullContent)
	if assistantMessage.Metadata == nil {
		assistantMessage.Metadata = &models.AssistantMetadata{}
	}
	assistantMessage.Metadata.LatencyMs = latencyMs
	assistantMessage.Metadata.AgentType = string(agentConfig.Type)

	if err := h.docDBClient.Messages().Add(ctx, assistantMessage); err == nil {
		_ = writer.WriteMessageComplete(assistantMessage)
	}
}

func (h *MessagesHandler) streamTitleGeneration(ctx context.Context, writer *sse.Writer, tenantID, conversationID, userContent, assistantContent, authToken string) {
	title, err := h.aiService.GenerateTitle(ctx, tenantID, userContent, assistantContent)
	if err != nil {
		log.Warn().Err(err).Str("conversation_id", conversationID).Msg("failed to generate AI title")
		return
	}

	_ = writer.WriteTitleGeneration(title)

	if authToken != "" {
		go func() {
			persistCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if err := h.platformClient.UpdateConversationTitle(persistCtx, tenantID, conversationID, title, authToken); err != nil {
				log.Warn().Err(err).Str("conversation_id", conversationID).Msg("failed to persist conversation title")
			}
		}()
	}
}

// convertFilesToFileInputs converts FileAttachment slice to agents.FileInput slice.
func convertFilesToFileInputs(files []FileAttachment) []agents.FileInput {
	if len(files) == 0 {
		return nil
	}

	result := make([]agents.FileInput, len(files))
	for i, f := range files {
		result[i] = agents.FileInput{
			Type:     f.Type,
			ImageURL: f.ImageURL,
			FileData: f.FileData,
			Filename: f.Filename,
			MimeType: f.MimeType,
			Detail:   f.Detail,
		}
	}
	return result
}

// convertFilesToAttachmentMetadata converts FileAttachment slice to AttachmentMetadata slice.
func convertFilesToAttachmentMetadata(files []FileAttachment) []models.AttachmentMetadata {
	if len(files) == 0 {
		return nil
	}

	result := make([]models.AttachmentMetadata, len(files))
	for i, f := range files {
		var fileSize int64
		if f.FileData != "" {
			fileSize = int64(len(f.FileData) * 3 / 4)
		} else if f.ImageURL != "" {
			idx := strings.Index(f.ImageURL, ",")
			if idx > 0 && idx < len(f.ImageURL)-1 {
				fileSize = int64(len(f.ImageURL[idx+1:]) * 3 / 4)
			}
		}
		result[i] = models.AttachmentMetadata{
			FileName:     f.Filename,
			FileType:     f.MimeType,
			FileSize:     fileSize,
			FileCategory: f.Type,
		}
	}
	return result
}

// extractConfigString extracts a string value from a config map by key.
func extractConfigString(config map[string]interface{}, key string) string {
	if config == nil {
		return ""
	}
	v, ok := config[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

// appendStatusTrace appends a new StatusTrace entry with the given type, name, and config data.
func appendStatusTrace(traces []models.StatusTrace, traceType, name string, config map[string]interface{}) []models.StatusTrace {
	trace := models.StatusTrace{
		Type:      traceType,
		Name:      name,
		Timestamp: time.Now().UTC(),
	}
	if len(config) > 0 {
		trace.Data = config
	}
	return append(traces, trace)
}

// appendToLastTrace appends content to the last StatusTrace entry's Content field.
func appendToLastTrace(traces []models.StatusTrace, content string) []models.StatusTrace {
	if len(traces) == 0 {
		return traces
	}
	traces[len(traces)-1].Content += content
	return traces
}
