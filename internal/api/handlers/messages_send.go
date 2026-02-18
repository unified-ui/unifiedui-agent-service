package handlers

import (
	"context"
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
// @Param request body SendMessageRequest true "Message content with applicationId"
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

		agentConfig, err = h.platformClient.GetAgentConfig(ctx, tenantCtx.TenantID, req.ApplicationID, conversationID, authToken)
		if err != nil {
			middleware.HandleError(c, errors.NewInternalError("failed to get agent configuration", err))
			return
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
		req.ApplicationID,
		tenantCtx.UserID,
		req.Message.Content,
		req.Message.Attachments,
		&models.MessageRequest{
			ApplicationID:  req.ApplicationID,
			ConversationID: req.ConversationID,
			Message: models.MessageRequestContent{
				Content:     req.Message.Content,
				Attachments: req.Message.Attachments,
			},
			InvokeConfig: models.MessageInvokeConfig{
				ChatHistoryMessageCount: req.InvokeConfig.ChatHistoryMessageCount,
			},
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
	} else {
		agentClients, createErr = h.agentFactory.CreateClients(agentConfig)
	}

	if createErr != nil {
		middleware.HandleError(c, errors.NewInternalError("failed to create agent clients", createErr))
		return
	}
	defer agentClients.Close()

	assistantMessage := models.NewAssistantMessage(
		tenantCtx.TenantID,
		conversationID,
		userMessageID,
		req.ApplicationID,
		"",
		models.MessageStatusPending,
	)
	assistantMessage.ID = assistantMessageID

	foundryAPIKey := c.GetHeader("X-Microsoft-Foundry-API-Key")
	authToken := middleware.GetToken(c)
	isFirstMessage := len(chatHistory) == 0
	if agentConfig.Type == platform.AgentTypeFoundry {
		count, err := h.docDBClient.Messages().CountByConversation(ctx, tenantCtx.TenantID, conversationID)
		if err == nil {
			isFirstMessage = count == 1
		}
	}
	files := convertFilesToFileInputs(req.Message.Files)
	h.handleStreamingResponse(c, tenantCtx, agentClients, agentConfig, userMessage, assistantMessage, chatHistory, req.ExtConversationID, foundryAPIKey, req.InvokeConfig.ContextData, authToken, isFirstMessage, files)
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
	if agentConfig.Type == platform.AgentTypeFoundry {
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
		writer.WriteStreamStart(assistantMessage.ID, userMessage.ConversationID)
		writer.WriteStreamError("STREAM_ERROR", errorMsg, err.Error())
		writer.WriteStreamEnd()
		h.saveFailedAssistantMessage(ctx, assistantMessage, errorMsg)
		writer.WriteMessageComplete(assistantMessage)
		return
	}
	defer streamReader.Close()

	startTime := time.Now()

	if agentConfig.Type == platform.AgentTypeFoundry {
		h.handleFoundryStreaming(ctx, writer, streamReader, tenantCtx, agentConfig, userMessage, assistantMessage, startTime)
		h.updateSessionCacheConfigOnly(ctx, tenantCtx, agentConfig, userMessage.ConversationID)

		if h.importService != nil && extConversationID != "" && foundryAPIKey != "" {
			h.enqueueFoundryTraceImport(tenantCtx, agentConfig, userMessage, extConversationID, foundryAPIKey)
		}
	} else {
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

	writer.WriteStreamStart(assistantMessage.ID, userMessage.ConversationID)

	for {
		select {
		case <-ctx.Done():
			streamReader.Close()
			writer.WriteStreamEnd()
			h.saveCancelledAssistantMessage(assistantMessage, fullContent, agentConfig, startTime)
			return executionID
		default:
		}

		chunk, err := streamReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			if ctx.Err() != nil {
				streamReader.Close()
				h.saveCancelledAssistantMessage(assistantMessage, fullContent, agentConfig, startTime)
				return executionID
			}
			errorMsg := "Stream error: " + err.Error()
			writer.WriteStreamError("STREAM_ERROR", errorMsg, err.Error())
			h.saveFailedAssistantMessage(ctx, assistantMessage, errorMsg)
			writer.WriteStreamEnd()
			writer.WriteMessageComplete(assistantMessage)
			return executionID
		}

		switch chunk.Type {
		case agents.ChunkTypeContent:
			fullContent += chunk.Content
			writer.WriteTextStream(chunk.Content)
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
				writer.WriteStreamError("CHUNK_ERROR", errorMsg, errorMsg)
				writer.WriteStreamEnd()
				h.saveFailedAssistantMessage(ctx, assistantMessage, errorMsg)
				writer.WriteMessageComplete(assistantMessage)
				return executionID
			}
		}
	}

	writer.WriteStreamEnd()

	latencyMs := time.Since(startTime).Milliseconds()

	assistantMessage.SetSuccess(fullContent)
	if assistantMessage.Metadata == nil {
		assistantMessage.Metadata = &models.AssistantMetadata{}
	}
	assistantMessage.Metadata.LatencyMs = latencyMs
	assistantMessage.Metadata.AgentType = string(agentConfig.Type)

	if err := h.docDBClient.Messages().Add(ctx, assistantMessage); err == nil {
		writer.WriteMessageComplete(assistantMessage)
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
	allMessages := []*models.Message{currentMessage}

	saveCurrentAndStartNew := func() {
		if currentContent != "" {
			currentMessage.SetSuccess(currentContent)
			h.saveAssistantMessageWithMetadata(ctx, currentMessage, agentConfig, startTime)
		}

		writer.WriteJSON(sse.EventMessage, &sse.StreamMessage{
			Type: "STREAM_NEW_MESSAGE",
		})

		currentMessage = models.NewAssistantMessage(
			tenantCtx.TenantID,
			userMessage.ConversationID,
			userMessage.ID,
			userMessage.ApplicationID,
			"",
			models.MessageStatusPending,
		)
		currentMessage.ID = generateMessageID()
		allMessages = append(allMessages, currentMessage)
		currentContent = ""

		writer.WriteStreamStart(currentMessage.ID, userMessage.ConversationID)
	}

	writer.WriteStreamStart(currentMessage.ID, userMessage.ConversationID)

	for {
		select {
		case <-ctx.Done():
			streamReader.Close()
			writer.WriteStreamEnd()
			h.saveCancelledAssistantMessage(currentMessage, currentContent, agentConfig, startTime)
			return
		default:
		}

		chunk, err := streamReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			if ctx.Err() != nil {
				streamReader.Close()
				h.saveCancelledAssistantMessage(currentMessage, currentContent, agentConfig, startTime)
				return
			}
			errorMsg := "Stream error: " + err.Error()
			writer.WriteStreamError("STREAM_ERROR", errorMsg, err.Error())
			h.saveFailedAssistantMessage(ctx, currentMessage, errorMsg)
			writer.WriteStreamEnd()
			writer.WriteMessageComplete(currentMessage)
			return
		}

		switch chunk.Type {
		case agents.ChunkTypeContent:
			currentContent += chunk.Content
			writer.WriteTextStream(chunk.Content)

		case agents.ChunkTypeNewMessage:
			saveCurrentAndStartNew()
			if chunk.Metadata != nil {
				currentMessage.Metadata = h.extractFoundryMetadata(chunk.Metadata)
			}

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
				writer.WriteStreamError("CHUNK_ERROR", errorMsg, errorMsg)
				writer.WriteStreamEnd()
				h.saveFailedAssistantMessage(ctx, currentMessage, errorMsg)
				writer.WriteMessageComplete(currentMessage)
				return
			}
		}
	}

	writer.WriteStreamEnd()

	currentMessage.SetSuccess(currentContent)
	savedMsg := h.saveAssistantMessageWithMetadata(ctx, currentMessage, agentConfig, startTime)
	if savedMsg != nil {
		writer.WriteMessageComplete(savedMsg)
	}
}

func (h *MessagesHandler) streamTitleGeneration(ctx context.Context, writer *sse.Writer, tenantID, conversationID, userContent, assistantContent, authToken string) {
	title, err := h.aiService.GenerateTitle(ctx, tenantID, userContent, assistantContent)
	if err != nil {
		log.Warn().Err(err).Str("conversation_id", conversationID).Msg("failed to generate AI title")
		return
	}

	writer.WriteTitleGeneration(title)

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
