// Package agents provides the agent client factory.
package agents

import (
	"context"
	"fmt"

	"github.com/unifiedui/agent-service/internal/services/agents/foundry"
	"github.com/unifiedui/agent-service/internal/services/agents/n8n"
	"github.com/unifiedui/agent-service/internal/services/platform"
)

// Factory creates agent clients based on configuration.
type Factory struct{}

// NewFactory creates a new agent factory.
func NewFactory() *Factory {
	return &Factory{}
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
	case platform.AgentTypeCopilot:
		return nil, fmt.Errorf("copilot agent type not yet implemented")
	case platform.AgentTypeCustom:
		return nil, fmt.Errorf("custom agent type not yet implemented")
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
		WorkflowClient: &foundryWorkflowAdapter{workflowClient},
		APIClient:      nil, // Foundry doesn't have a separate API client
		Config:         config,
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
		workflowClient.Close()
		return nil, fmt.Errorf("failed to create N8N API client: %w", err)
	}

	return &AgentClients{
		WorkflowClient: &n8nWorkflowAdapter{workflowClient},
		APIClient:      &n8nAPIAdapter{apiClient},
		Config:         config,
	}, nil
}

// n8nWorkflowAdapter adapts n8n.ChatWorkflowClient to agents.WorkflowClient interface.
type n8nWorkflowAdapter struct {
	client *n8n.ChatWorkflowClient
}

func (a *n8nWorkflowAdapter) Invoke(ctx context.Context, req *InvokeRequest) (*InvokeResponse, error) {
	n8nReq := &n8n.InvokeRequest{
		ConversationID: req.ConversationID,
		Message:        req.Message,
		SessionID:      req.SessionID,
		ChatHistory:    req.ChatHistory,
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
	n8nReq := &n8n.InvokeRequest{
		ConversationID: req.ConversationID,
		Message:        req.Message,
		SessionID:      req.SessionID,
		ChatHistory:    req.ChatHistory,
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
	n8nReq := &n8n.InvokeRequest{
		ConversationID: req.ConversationID,
		Message:        req.Message,
		SessionID:      req.SessionID,
		ChatHistory:    req.ChatHistory,
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
	client *foundry.WorkflowClient
}

func (a *foundryWorkflowAdapter) Invoke(ctx context.Context, req *InvokeRequest) (*InvokeResponse, error) {
	foundryReq := &foundry.InvokeRequest{
		ExtConversationID: req.ConversationID,
		Message:           req.Message,
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
	foundryReq := &foundry.InvokeRequest{
		ExtConversationID: req.ConversationID,
		Message:           req.Message,
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
	foundryReq := &foundry.InvokeRequest{
		ExtConversationID: req.ConversationID,
		Message:           req.Message,
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
		Error:       foundryChunk.Error,
	}
}
