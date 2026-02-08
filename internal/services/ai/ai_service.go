// Package ai provides the AI service for LLM interactions.
package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/unifiedui/agent-service/internal/services/platform"
)

// aiService implements the Service interface.
type aiService struct {
	platformClient platform.Client
}

// NewService creates a new AI service.
func NewService(platformClient platform.Client) Service {
	return &aiService{
		platformClient: platformClient,
	}
}

// GenerateTitle generates a concise conversation title from the first message exchange.
func (s *aiService) GenerateTitle(ctx context.Context, tenantID, userMessage, assistantResponse string) (string, error) {
	llmClient, err := s.getLLMClientForPurpose(ctx, tenantID, "CONVERSATION_TITLE_GENERATION", "LLM_MODEL")
	if err != nil {
		return "", fmt.Errorf("failed to get LLM client: %w", err)
	}

	messages := BuildTitleGenerationMessages(userMessage, assistantResponse)

	result, err := llmClient.ChatCompletion(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	title := result.Content
	if len(title) > 50 {
		title = title[:50]
	}

	return title, nil
}

// GenerateDescription generates or improves a description for an entity.
func (s *aiService) GenerateDescription(ctx context.Context, tenantID, entityType, entityName, existingDescription string, entityContext map[string]interface{}) (string, error) {
	llmClient, err := s.getLLMClientForPurpose(ctx, tenantID, "DESCRIPTION_GENERATION", "LLM_MODEL")
	if err != nil {
		return "", fmt.Errorf("failed to get LLM client: %w", err)
	}

	messages := BuildDescriptionGenerationMessages(entityType, entityName, existingDescription, entityContext)

	result, err := llmClient.ChatCompletion(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	return result.Content, nil
}

// AnalyzeTrace analyzes a failed trace node and returns error analysis.
func (s *aiService) AnalyzeTrace(ctx context.Context, tenantID string, request AnalyzeTraceInput) (string, error) {
	llmClient, err := s.getLLMClientForPurpose(ctx, tenantID, "TRACE_ANALYSIS", "LLM_MODEL")
	if err != nil {
		return "", fmt.Errorf("failed to get LLM client: %w", err)
	}

	inputTOML := ""
	if request.Input != nil {
		inputTOML = ToTOML(request.Input)
	}
	outputTOML := ""
	if request.Output != nil {
		outputTOML = ToTOML(request.Output)
	}

	messages := BuildTraceAnalysisMessages(request.NodeName, request.NodeType, request.Error, inputTOML, outputTOML)

	result, err := llmClient.ChatCompletion(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	return result.Content, nil
}

// SummarizeTrace summarizes trace nodes at the specified detail level.
func (s *aiService) SummarizeTrace(ctx context.Context, tenantID string, request SummarizeTraceInput) (string, error) {
	llmClient, err := s.getLLMClientForPurpose(ctx, tenantID, "TRACE_ANALYSIS", "LLM_MODEL")
	if err != nil {
		return "", fmt.Errorf("failed to get LLM client: %w", err)
	}

	nodesTOML := SliceToTOML(request.Nodes, "nodes")

	messages := BuildTraceSummarizeMessages(request.DetailLevel, nodesTOML)

	result, err := llmClient.ChatCompletion(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	return result.Content, nil
}

// TestModel tests an LLM model configuration by sending a simple ping.
func (s *aiService) TestModel(ctx context.Context, provider string, config map[string]interface{}, credentialSecret map[string]interface{}) (*TestModelResult, error) {
	llmClient, err := NewLLMClient(provider, config, credentialSecret)
	if err != nil {
		return &TestModelResult{
			Success:        false,
			Message:        fmt.Sprintf("Failed to create LLM client: %s", err.Error()),
			ResponseTimeMs: 0,
		}, nil
	}

	messages := BuildTestModelMessages()

	start := time.Now()
	_, err = llmClient.ChatCompletion(ctx, messages)
	responseTime := time.Since(start).Milliseconds()

	if err != nil {
		return &TestModelResult{
			Success:        false,
			Message:        fmt.Sprintf("Model test failed: %s", err.Error()),
			ResponseTimeMs: responseTime,
		}, nil
	}

	return &TestModelResult{
		Success:        true,
		Message:        "Model responded successfully",
		ResponseTimeMs: responseTime,
	}, nil
}

// GetCapabilities returns the available AI capabilities for a tenant.
func (s *aiService) GetCapabilities(ctx context.Context, tenantID string) (*Capabilities, error) {
	capabilities := &Capabilities{}

	purposeGroups := map[string]*bool{
		"CONVERSATION_TITLE_GENERATION": &capabilities.TitleGeneration,
		"DESCRIPTION_GENERATION":        &capabilities.DescriptionGeneration,
		"TRACE_ANALYSIS":                &capabilities.TraceAnalysis,
		"CONVERSATION_SUMMARIZATION":    &capabilities.Summarization,
	}

	for purpose, field := range purposeGroups {
		models, err := s.platformClient.GetAIModelsByPurpose(ctx, tenantID, purpose, "LLM_MODEL")
		if err != nil {
			log.Warn().Err(err).Str("purpose", purpose).Msg("failed to check AI models for purpose")
			continue
		}
		*field = len(models) > 0
	}

	return capabilities, nil
}

func (s *aiService) getLLMClientForPurpose(ctx context.Context, tenantID, purposeGroup, modelType string) (LLMClient, error) {
	models, err := s.platformClient.GetAIModelsByPurpose(ctx, tenantID, purposeGroup, modelType)
	if err != nil {
		return nil, fmt.Errorf("failed to get AI models: %w", err)
	}

	if len(models) == 0 {
		return nil, fmt.Errorf("no active AI models configured for purpose: %s", purposeGroup)
	}

	model := models[0]

	llmClient, err := NewLLMClient(model.Provider, model.Config, model.CredentialSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client for provider %s: %w", model.Provider, err)
	}

	return llmClient, nil
}
