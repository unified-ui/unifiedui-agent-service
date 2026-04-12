// Package agents provides the agent client factory.
package agents

import (
	"context"
	"fmt"

	"github.com/unifiedui/agent-service/internal/config"
	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/pkg/contextformat"
	"github.com/unifiedui/agent-service/internal/services/agents/foundry"
	"github.com/unifiedui/agent-service/internal/services/agents/llm"
	"github.com/unifiedui/agent-service/internal/services/agents/n8n"
	"github.com/unifiedui/agent-service/internal/services/agents/react"
	"github.com/unifiedui/agent-service/internal/services/agents/restapi"
	"github.com/unifiedui/agent-service/internal/services/platform"
)

// Factory creates agent clients based on configuration.
type Factory struct {
	reactFactory *react.Factory
}

// NewFactory creates a new agent factory.
func NewFactory() *Factory {
	return &Factory{}
}

// NewFactoryWithReact creates a new agent factory with ReACT service support.
func NewFactoryWithReact(reactCfg config.ReactServiceConfig, serviceKey string) *Factory {
	return &Factory{
		reactFactory: react.NewFactory(reactCfg, serviceKey),
	}
}

// CreateClients creates the appropriate agent clients based on the configuration.
func (f *Factory) CreateClients(config *platform.AgentConfig) (*AgentClients, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	switch config.Type {
	case platform.AgentTypeN8N:
		return f.createN8NClients(config)
	case platform.AgentTypeFoundry:
		return nil, fmt.Errorf("foundry requires API token - use CreateFoundryClients instead")
	case platform.AgentTypeReactAgent:
		return f.createReActClients(config)
	case platform.AgentTypeCopilot:
		return nil, fmt.Errorf("copilot agent type not yet implemented")
	case platform.AgentTypeCustom:
		return nil, fmt.Errorf("custom agent type not yet implemented")
	case platform.AgentTypeRestAPI:
		return nil, fmt.Errorf("REST API requires auth token - use CreateRestAPIClients instead")
	default:
		return nil, fmt.Errorf("unsupported agent type: %s", config.Type)
	}
}

// CreateFoundryClients creates Microsoft Foundry agent clients with the provided API token.
func (f *Factory) CreateFoundryClients(config *platform.AgentConfig, apiToken string) (*AgentClients, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if apiToken == "" {
		return nil, fmt.Errorf("API token is required for Foundry agents")
	}

	foundryFactory := foundry.NewFactory()

	workflowClient, err := foundryFactory.CreateWorkflowClient(config, apiToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create Foundry workflow client: %w", err)
	}

	return &AgentClients{
		WorkflowClient: &foundryWorkflowAdapter{
			client:        workflowClient,
			fileConverter: foundry.NewFileConverter(),
		},
		APIClient: nil, // Foundry doesn't have a separate API client
		Config:    config,
	}, nil
}

// createN8NClients creates N8N-specific clients.
func (f *Factory) createN8NClients(config *platform.AgentConfig) (*AgentClients, error) {
	n8nFactory := n8n.NewFactory()

	// Create workflow client
	workflowClient, err := n8nFactory.CreateWorkflowClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create N8N workflow client: %w", err)
	}

	// Create API client
	apiClient, err := n8nFactory.CreateAPIClient(config)
	if err != nil {
		// Clean up workflow client if API client creation fails
		_ = workflowClient.Close()
		return nil, fmt.Errorf("failed to create N8N API client: %w", err)
	}

	return &AgentClients{
		WorkflowClient: &n8nWorkflowAdapter{
			client:        workflowClient,
			fileConverter: n8n.NewFileConverter(),
		},
		APIClient: &n8nAPIAdapter{apiClient},
		Config:    config,
	}, nil
}

// n8nWorkflowAdapter adapts n8n.ChatWorkflowClient to agents.WorkflowClient interface.
type n8nWorkflowAdapter struct {
	client        *n8n.ChatWorkflowClient
	fileConverter *n8n.FileConverter
}

// toN8NFileInputs converts agents.FileInput to n8n.FileInput.
func toN8NFileInputs(files []FileInput) []n8n.FileInput {
	result := make([]n8n.FileInput, len(files))
	for i, f := range files {
		result[i] = n8n.FileInput{
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

func (a *n8nWorkflowAdapter) Invoke(ctx context.Context, req *InvokeRequest) (*InvokeResponse, error) {
	// Prepend context data to message if present
	message := contextformat.PrependContextToMessage(req.ContextData, req.Message)

	// Convert files if present
	var input interface{}
	if len(req.Files) > 0 {
		n8nFiles := toN8NFileInputs(req.Files)
		converted, err := a.fileConverter.ConvertFiles(message, n8nFiles)
		if err != nil {
			return nil, err
		}
		input = converted
	}

	n8nReq := &n8n.InvokeRequest{
		ConversationID: req.ConversationID,
		Message:        message,
		SessionID:      req.SessionID,
		ChatHistory:    req.ChatHistory,
		Input:          input,
	}

	resp, err := a.client.Invoke(ctx, n8nReq)
	if err != nil {
		return nil, err
	}

	return &InvokeResponse{
		Content:     resp.Content,
		ExecutionID: resp.ExecutionID,
		SessionID:   resp.SessionID,
		Metadata:    resp.Metadata,
	}, nil
}

func (a *n8nWorkflowAdapter) InvokeStream(ctx context.Context, req *InvokeRequest) (<-chan *StreamChunk, error) {
	// Prepend context data to message if present
	message := contextformat.PrependContextToMessage(req.ContextData, req.Message)

	// Convert files if present
	var input interface{}
	if len(req.Files) > 0 {
		n8nFiles := toN8NFileInputs(req.Files)
		converted, err := a.fileConverter.ConvertFiles(message, n8nFiles)
		if err != nil {
			return nil, err
		}
		input = converted
	}

	n8nReq := &n8n.InvokeRequest{
		ConversationID: req.ConversationID,
		Message:        message,
		SessionID:      req.SessionID,
		ChatHistory:    req.ChatHistory,
		Input:          input,
	}

	n8nCh, err := a.client.InvokeStream(ctx, n8nReq)
	if err != nil {
		return nil, err
	}

	ch := make(chan *StreamChunk, 100)
	go func() {
		defer close(ch)
		for n8nChunk := range n8nCh {
			ch <- convertN8NChunk(n8nChunk)
		}
	}()

	return ch, nil
}

func (a *n8nWorkflowAdapter) InvokeStreamReader(ctx context.Context, req *InvokeRequest) (StreamReader, error) {
	// Prepend context data to message if present
	message := contextformat.PrependContextToMessage(req.ContextData, req.Message)

	// Convert files if present
	var input interface{}
	if len(req.Files) > 0 {
		n8nFiles := toN8NFileInputs(req.Files)
		converted, err := a.fileConverter.ConvertFiles(message, n8nFiles)
		if err != nil {
			return nil, err
		}
		input = converted
	}

	n8nReq := &n8n.InvokeRequest{
		ConversationID: req.ConversationID,
		Message:        message,
		SessionID:      req.SessionID,
		ChatHistory:    req.ChatHistory,
		Input:          input,
	}

	reader, err := a.client.InvokeStreamReader(ctx, n8nReq)
	if err != nil {
		return nil, err
	}

	return &n8nStreamReaderAdapter{reader}, nil
}

func (a *n8nWorkflowAdapter) Close() error {
	return a.client.Close()
}

// n8nStreamReaderAdapter adapts n8n.StreamReader to agents.StreamReader.
type n8nStreamReaderAdapter struct {
	reader n8n.StreamReader
}

func (a *n8nStreamReaderAdapter) Read() (*StreamChunk, error) {
	chunk, err := a.reader.Read()
	if err != nil {
		return nil, err
	}
	return convertN8NChunk(chunk), nil
}

func (a *n8nStreamReaderAdapter) Close() error {
	return a.reader.Close()
}

// convertN8NChunk converts n8n.StreamChunk to agents.StreamChunk.
func convertN8NChunk(n8nChunk *n8n.StreamChunk) *StreamChunk {
	return &StreamChunk{
		Type:        ChunkType(n8nChunk.Type),
		Content:     n8nChunk.Content,
		ExecutionID: n8nChunk.ExecutionID,
		Metadata:    n8nChunk.Metadata,
		Error:       n8nChunk.Error,
	}
}

// n8nAPIAdapter adapts n8n.APIClient to agents.APIClient interface.
type n8nAPIAdapter struct {
	client *n8n.APIClient
}

func (a *n8nAPIAdapter) GetExecution(ctx context.Context, executionID string) (*ExecutionInfo, error) {
	info, err := a.client.GetExecution(ctx, executionID)
	if err != nil {
		return nil, err
	}

	return &ExecutionInfo{
		ID:        info.ID,
		Status:    info.Status,
		StartedAt: info.StartedAt,
		StoppedAt: info.StoppedAt,
		Data:      info.Data,
	}, nil
}

func (a *n8nAPIAdapter) GetExecutionsBySession(ctx context.Context, sessionID string) ([]*ExecutionInfo, error) {
	infos, err := a.client.GetExecutionsBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	result := make([]*ExecutionInfo, len(infos))
	for i, info := range infos {
		result[i] = &ExecutionInfo{
			ID:        info.ID,
			Status:    info.Status,
			StartedAt: info.StartedAt,
			StoppedAt: info.StoppedAt,
			Data:      info.Data,
		}
	}

	return result, nil
}

func (a *n8nAPIAdapter) Close() error {
	return a.client.Close()
}

// foundryWorkflowAdapter adapts foundry.WorkflowClient to agents.WorkflowClient interface.
type foundryWorkflowAdapter struct {
	client        *foundry.WorkflowClient
	fileConverter *foundry.FileConverter
}

// toFoundryFileInputs converts agents.FileInput to foundry.FileInput.
func toFoundryFileInputs(files []FileInput) []foundry.FileInput {
	result := make([]foundry.FileInput, len(files))
	for i, f := range files {
		result[i] = foundry.FileInput{
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

func (a *foundryWorkflowAdapter) Invoke(ctx context.Context, req *InvokeRequest) (*InvokeResponse, error) {
	// Prepend context data to message if present
	message := contextformat.PrependContextToMessage(req.ContextData, req.Message)

	// Convert files if present
	var input interface{}
	if len(req.Files) > 0 {
		foundryFiles := toFoundryFileInputs(req.Files)
		converted, err := a.fileConverter.ConvertFiles(message, foundryFiles)
		if err != nil {
			return nil, err
		}
		input = converted
	}

	foundryReq := &foundry.InvokeRequest{
		ExtConversationID: req.ConversationID,
		Message:           message,
		Input:             input,
	}

	resp, err := a.client.Invoke(ctx, foundryReq)
	if err != nil {
		return nil, err
	}

	return &InvokeResponse{
		Content:     resp.Content,
		ExecutionID: resp.ExecutionID,
		SessionID:   resp.SessionID,
		Metadata:    resp.Metadata,
	}, nil
}

func (a *foundryWorkflowAdapter) InvokeStream(ctx context.Context, req *InvokeRequest) (<-chan *StreamChunk, error) {
	// Prepend context data to message if present
	message := contextformat.PrependContextToMessage(req.ContextData, req.Message)

	// Convert files if present
	var input interface{}
	if len(req.Files) > 0 {
		foundryFiles := toFoundryFileInputs(req.Files)
		converted, err := a.fileConverter.ConvertFiles(message, foundryFiles)
		if err != nil {
			return nil, err
		}
		input = converted
	}

	foundryReq := &foundry.InvokeRequest{
		ExtConversationID: req.ConversationID,
		Message:           message,
		Input:             input,
	}

	foundryCh, err := a.client.InvokeStream(ctx, foundryReq)
	if err != nil {
		return nil, err
	}

	ch := make(chan *StreamChunk, 100)
	go func() {
		defer close(ch)
		for foundryChunk := range foundryCh {
			ch <- convertFoundryChunk(foundryChunk)
		}
	}()

	return ch, nil
}

func (a *foundryWorkflowAdapter) InvokeStreamReader(ctx context.Context, req *InvokeRequest) (StreamReader, error) {
	// Prepend context data to message if present
	message := contextformat.PrependContextToMessage(req.ContextData, req.Message)

	// Convert files if present
	var input interface{}
	if len(req.Files) > 0 {
		foundryFiles := toFoundryFileInputs(req.Files)
		converted, err := a.fileConverter.ConvertFiles(message, foundryFiles)
		if err != nil {
			return nil, err
		}
		input = converted
	}

	foundryReq := &foundry.InvokeRequest{
		ExtConversationID: req.ConversationID,
		Message:           message,
		Input:             input,
	}

	reader, err := a.client.InvokeStreamReader(ctx, foundryReq)
	if err != nil {
		return nil, err
	}

	return &foundryStreamReaderAdapter{reader}, nil
}

func (a *foundryWorkflowAdapter) Close() error {
	return a.client.Close()
}

// foundryStreamReaderAdapter adapts foundry.StreamReader to agents.StreamReader.
type foundryStreamReaderAdapter struct {
	reader foundry.StreamReader
}

func (a *foundryStreamReaderAdapter) Read() (*StreamChunk, error) {
	chunk, err := a.reader.Read()
	if err != nil {
		return nil, err
	}
	return convertFoundryChunk(chunk), nil
}

func (a *foundryStreamReaderAdapter) Close() error {
	return a.reader.Close()
}

// convertFoundryChunk converts foundry.StreamChunk to agents.StreamChunk.
func convertFoundryChunk(foundryChunk *foundry.StreamChunk) *StreamChunk {
	return &StreamChunk{
		Type:        ChunkType(foundryChunk.Type),
		Content:     foundryChunk.Content,
		ExecutionID: foundryChunk.ExecutionID,
		Metadata:    foundryChunk.Metadata,
		Config:      foundryChunk.Config,
		Error:       foundryChunk.Error,
	}
}

// CreateRestAPIClients creates REST API agent clients with the user token for auth forwarding.
func (f *Factory) CreateRestAPIClients(config *platform.AgentConfig, userToken string) (*AgentClients, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	restFactory := restapi.NewFactory()

	workflowClient, err := restFactory.CreateWorkflowClient(config, userToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create REST API workflow client: %w", err)
	}

	return &AgentClients{
		WorkflowClient: &restAPIWorkflowAdapter{
			client: workflowClient,
		},
		APIClient: nil,
		Config:    config,
	}, nil
}

// CreateLLMClients creates LLM agent clients for direct model chat streaming.
func (f *Factory) CreateLLMClients(config *platform.AgentConfig) (*AgentClients, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	aiModel := config.Settings.AIModel
	if aiModel == nil {
		return nil, fmt.Errorf("LLM agent requires a resolved AI model in config")
	}

	streamClient, err := llm.NewStreamingClient(aiModel.Provider, aiModel.Config, aiModel.CredentialSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM streaming client: %w", err)
	}

	return &AgentClients{
		WorkflowClient: &llmWorkflowAdapter{client: streamClient, settings: config.Settings},
		APIClient:      nil,
		Config:         config,
	}, nil
}

// createReActClients creates ReACT agent clients.
func (f *Factory) createReActClients(config *platform.AgentConfig) (*AgentClients, error) {
	if f.reactFactory == nil {
		return nil, fmt.Errorf("ReACT service not configured - use NewFactoryWithReact")
	}

	workflowClient, err := f.reactFactory.CreateWorkflowClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create ReACT workflow client: %w", err)
	}

	return &AgentClients{
		WorkflowClient: &reactWorkflowAdapter{
			client: workflowClient,
			config: config,
		},
		APIClient: nil,
		Config:    config,
	}, nil
}

// reactWorkflowAdapter adapts react.WorkflowClient to agents.WorkflowClient interface.
type reactWorkflowAdapter struct {
	client *react.WorkflowClient
	config *platform.AgentConfig
}

func (a *reactWorkflowAdapter) Invoke(ctx context.Context, req *InvokeRequest) (*InvokeResponse, error) {
	reactReq := a.buildReActRequest(req)

	content, err := a.client.Invoke(ctx, reactReq)
	if err != nil {
		return nil, err
	}

	return &InvokeResponse{
		Content: content,
	}, nil
}

func (a *reactWorkflowAdapter) InvokeStream(ctx context.Context, req *InvokeRequest) (<-chan *StreamChunk, error) {
	reader, err := a.InvokeStreamReader(ctx, req)
	if err != nil {
		return nil, err
	}

	ch := make(chan *StreamChunk, 100)
	go func() {
		defer close(ch)
		for {
			chunk, readErr := reader.Read()
			if readErr != nil {
				_ = reader.Close()
				return
			}
			ch <- chunk
		}
	}()

	return ch, nil
}

func (a *reactWorkflowAdapter) InvokeStreamReader(ctx context.Context, req *InvokeRequest) (StreamReader, error) {
	reactReq := a.buildReActRequest(req)

	reader, err := a.client.InvokeStreamReader(ctx, reactReq)
	if err != nil {
		return nil, err
	}

	return &reactStreamReaderAdapter{reader: reader}, nil
}

func (a *reactWorkflowAdapter) Close() error {
	return a.client.Close()
}

// buildReActRequest converts the generic InvokeRequest + AgentConfig into a react.InvokeRequest.
func (a *reactWorkflowAdapter) buildReActRequest(req *InvokeRequest) *react.InvokeRequest {
	message := contextformat.PrependContextToMessage(req.ContextData, req.Message)

	aiModels := make([]react.AIModelConfig, 0)
	tools := make([]react.ToolDefinition, 0)

	if a.config != nil {
		for _, m := range a.config.Settings.AIModels {
			am := react.AIModelConfig{
				Provider: m.Provider,
			}
			if v, ok := m.Config["model_name"]; ok {
				am.ModelName, _ = v.(string)
			}
			if v, ok := m.Config["base_url"]; ok {
				am.BaseURL, _ = v.(string)
			}
			if v, ok := m.Config["endpoint"]; ok {
				am.Endpoint, _ = v.(string)
			}
			if v, ok := m.Config["api_version"]; ok {
				am.APIVersion, _ = v.(string)
			}
			if v, ok := m.Config["deployment_name"]; ok {
				am.DeploymentName, _ = v.(string)
			}
			if v, ok := m.Config["organization"]; ok {
				am.Organization, _ = v.(string)
			}
			if m.CredentialSecret != nil {
				if apiKey, ok := m.CredentialSecret["api_key"]; ok {
					am.APIKey, _ = apiKey.(string)
				}
			}
			aiModels = append(aiModels, am)
		}
	}

	if a.config != nil && a.config.Settings.Tools != nil {
		for _, t := range a.config.Settings.Tools {
			td := react.ToolDefinition{
				ID:          t.ID,
				Name:        t.Name,
				Description: t.Description,
				Type:        t.Type,
				Config:      t.Config,
				IsActive:    t.IsActive,
			}
			if t.Credentials != nil {
				td.Credential = &react.ToolCredential{
					ID:     t.Credentials.ID,
					Type:   string(t.Credentials.Type),
					Secret: t.Credentials.GetSecretAsString(),
				}
			}
			tools = append(tools, td)
		}
	}

	history := req.ChatHistory
	if history == nil {
		history = make([]models.ChatHistoryEntry, 0)
	}

	return &react.InvokeRequest{
		TenantID:       a.config.TenantID,
		ChatAgentID:    a.config.ChatAgentID,
		ConversationID: req.ConversationID,
		Message:        message,
		History:        history,
		AgentConfig: react.AgentConfigPayload{
			ReactAgentID:      a.config.Settings.ReActAgentID,
			SystemPrompt:      a.config.Settings.SystemPrompt,
			AIModels:          aiModels,
			Tools:             tools,
			MultiAgentEnabled: false,
		},
	}
}

// reactStreamReaderAdapter adapts react.StreamReader to agents.StreamReader.
type reactStreamReaderAdapter struct {
	reader react.StreamReader
}

func (a *reactStreamReaderAdapter) Read() (*StreamChunk, error) {
	chunk, err := a.reader.Read()
	if err != nil {
		return nil, err
	}
	return convertReActChunk(chunk), nil
}

func (a *reactStreamReaderAdapter) Close() error {
	return a.reader.Close()
}

// convertReActChunk converts react.StreamChunk to agents.StreamChunk.
func convertReActChunk(reactChunk *react.StreamChunk) *StreamChunk {
	return &StreamChunk{
		Type:    ChunkType(reactChunk.Type),
		Content: reactChunk.Content,
		Config:  reactChunk.Config,
		Error:   reactChunk.Error,
	}
}

// restAPIWorkflowAdapter adapts restapi.WorkflowClient to agents.WorkflowClient interface.
type restAPIWorkflowAdapter struct {
	client *restapi.WorkflowClient
}

func (a *restAPIWorkflowAdapter) Invoke(ctx context.Context, req *InvokeRequest) (*InvokeResponse, error) {
	message := contextformat.PrependContextToMessage(req.ContextData, req.Message)

	reader, err := a.client.InvokeStreamReader(ctx, req.ConversationID, req.SessionID, message, req.ChatHistory)
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()

	var fullContent string
	for {
		chunk, readErr := reader.Read()
		if readErr != nil {
			break
		}
		if chunk.Type == restapi.ChunkTypeContent {
			fullContent += chunk.Content
		}
	}

	return &InvokeResponse{
		Content: fullContent,
	}, nil
}

func (a *restAPIWorkflowAdapter) InvokeStream(ctx context.Context, req *InvokeRequest) (<-chan *StreamChunk, error) {
	reader, err := a.InvokeStreamReader(ctx, req)
	if err != nil {
		return nil, err
	}

	ch := make(chan *StreamChunk, 100)
	go func() {
		defer close(ch)
		for {
			chunk, readErr := reader.Read()
			if readErr != nil {
				_ = reader.Close()
				return
			}
			ch <- chunk
		}
	}()

	return ch, nil
}

func (a *restAPIWorkflowAdapter) InvokeStreamReader(ctx context.Context, req *InvokeRequest) (StreamReader, error) {
	message := contextformat.PrependContextToMessage(req.ContextData, req.Message)

	reader, err := a.client.InvokeStreamReader(ctx, req.ConversationID, req.SessionID, message, req.ChatHistory)
	if err != nil {
		return nil, err
	}

	return &restAPIStreamReaderAdapter{reader: reader}, nil
}

func (a *restAPIWorkflowAdapter) Close() error {
	return a.client.Close()
}

// CreateConversation calls the external conversation creation endpoint.
func (a *restAPIWorkflowAdapter) CreateConversation(ctx context.Context) (string, error) {
	return a.client.CreateConversation(ctx)
}

// restAPIStreamReaderAdapter adapts restapi.streamReader to agents.StreamReader.
type restAPIStreamReaderAdapter struct {
	reader *restapi.StreamReader
}

func (a *restAPIStreamReaderAdapter) Read() (*StreamChunk, error) {
	chunk, err := a.reader.Read()
	if err != nil {
		return nil, err
	}
	return convertRestAPIChunk(chunk), nil
}

func (a *restAPIStreamReaderAdapter) Close() error {
	return a.reader.Close()
}

// convertRestAPIChunk converts restapi.StreamChunk to agents.StreamChunk.
func convertRestAPIChunk(restChunk *restapi.StreamChunk) *StreamChunk {
	return &StreamChunk{
		Type:    ChunkType(restChunk.Type),
		Content: restChunk.Content,
		Config:  restChunk.Config,
		Error:   restChunk.Error,
	}
}
