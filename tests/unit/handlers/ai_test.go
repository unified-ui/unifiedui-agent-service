// Package handlers_test provides unit tests for the API handlers.
package handlers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/unifiedui/agent-service/internal/api/dto"
	"github.com/unifiedui/agent-service/internal/api/handlers"
	"github.com/unifiedui/agent-service/internal/services/ai"
	"github.com/unifiedui/agent-service/tests/mocks"
	"github.com/unifiedui/agent-service/tests/testutils"
)

const testTenantID = "tenant-test-123"

// --- GenerateDescription Tests ---

func TestAIHandler_GenerateDescription_Success(t *testing.T) {
	mockAI := new(mocks.MockAIService)
	mockPlatform := new(mocks.MockPlatformClient)

	mockAI.On("GenerateDescription",
		mock.Anything, testTenantID, "application", "My App", "", mock.Anything,
	).Return("A powerful application for managing workflows.", nil)

	handler := handlers.NewAIHandler(mockAI, mockPlatform)
	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/ai/generate-description", handler.GenerateDescription)

	body := dto.GenerateDescriptionRequest{
		EntityType: "application",
		EntityName: "My App",
	}

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/ai/generate-description", testTenantID),
		body, nil)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var resp dto.GenerateDescriptionResponse
	testutils.ParseJSONResponse(t, w, &resp)
	assert.Equal(t, "A powerful application for managing workflows.", resp.Description)

	mockAI.AssertExpectations(t)
}

func TestAIHandler_GenerateDescription_WithExistingDescription(t *testing.T) {
	mockAI := new(mocks.MockAIService)
	mockPlatform := new(mocks.MockPlatformClient)

	mockAI.On("GenerateDescription",
		mock.Anything, testTenantID, "tool", "Search Tool", "does web searches", mock.Anything,
	).Return("An AI-powered web search tool for retrieving real-time information.", nil)

	handler := handlers.NewAIHandler(mockAI, mockPlatform)
	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/ai/generate-description", handler.GenerateDescription)

	body := dto.GenerateDescriptionRequest{
		EntityType:          "tool",
		EntityName:          "Search Tool",
		ExistingDescription: "does web searches",
	}

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/ai/generate-description", testTenantID),
		body, nil)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var resp dto.GenerateDescriptionResponse
	testutils.ParseJSONResponse(t, w, &resp)
	assert.Contains(t, resp.Description, "search")

	mockAI.AssertExpectations(t)
}

func TestAIHandler_GenerateDescription_MissingEntityType(t *testing.T) {
	mockAI := new(mocks.MockAIService)
	mockPlatform := new(mocks.MockPlatformClient)

	handler := handlers.NewAIHandler(mockAI, mockPlatform)
	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/ai/generate-description", handler.GenerateDescription)

	body := map[string]interface{}{
		"entity_name": "My App",
	}

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/ai/generate-description", testTenantID),
		body, nil)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestAIHandler_GenerateDescription_MissingEntityName(t *testing.T) {
	mockAI := new(mocks.MockAIService)
	mockPlatform := new(mocks.MockPlatformClient)

	handler := handlers.NewAIHandler(mockAI, mockPlatform)
	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/ai/generate-description", handler.GenerateDescription)

	body := map[string]interface{}{
		"entity_type": "application",
	}

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/ai/generate-description", testTenantID),
		body, nil)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestAIHandler_GenerateDescription_ServiceError(t *testing.T) {
	mockAI := new(mocks.MockAIService)
	mockPlatform := new(mocks.MockPlatformClient)

	mockAI.On("GenerateDescription",
		mock.Anything, testTenantID, "application", "My App", "", mock.Anything,
	).Return("", fmt.Errorf("no active AI models configured"))

	handler := handlers.NewAIHandler(mockAI, mockPlatform)
	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/ai/generate-description", handler.GenerateDescription)

	body := dto.GenerateDescriptionRequest{
		EntityType: "application",
		EntityName: "My App",
	}

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/ai/generate-description", testTenantID),
		body, nil)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)

	mockAI.AssertExpectations(t)
}

func TestAIHandler_GenerateDescription_EmptyBody(t *testing.T) {
	mockAI := new(mocks.MockAIService)
	mockPlatform := new(mocks.MockPlatformClient)

	handler := handlers.NewAIHandler(mockAI, mockPlatform)
	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/ai/generate-description", handler.GenerateDescription)

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/ai/generate-description", testTenantID),
		nil, nil)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

// --- AnalyzeTrace Tests ---

func TestAIHandler_AnalyzeTrace_Success(t *testing.T) {
	mockAI := new(mocks.MockAIService)
	mockPlatform := new(mocks.MockPlatformClient)

	mockAI.On("AnalyzeTrace",
		mock.Anything, testTenantID, mock.MatchedBy(func(req ai.AnalyzeTraceInput) bool {
			return req.NodeName == "HTTP Request" && req.Error == "Connection refused"
		}),
	).Return("## Error Analysis\n\n**Root Cause:** Connection refused...", nil)

	handler := handlers.NewAIHandler(mockAI, mockPlatform)
	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/ai/analyze-trace", handler.AnalyzeTrace)

	body := dto.AnalyzeTraceRequest{
		TraceID:  "trace-123",
		NodeName: "HTTP Request",
		NodeType: "http",
		Error:    "Connection refused",
		Input:    map[string]interface{}{"url": "https://api.example.com"},
	}

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/ai/analyze-trace", testTenantID),
		body, nil)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var resp dto.AnalyzeTraceResponse
	testutils.ParseJSONResponse(t, w, &resp)
	assert.Contains(t, resp.Analysis, "Error Analysis")

	mockAI.AssertExpectations(t)
}

func TestAIHandler_AnalyzeTrace_MissingError(t *testing.T) {
	mockAI := new(mocks.MockAIService)
	mockPlatform := new(mocks.MockPlatformClient)

	handler := handlers.NewAIHandler(mockAI, mockPlatform)
	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/ai/analyze-trace", handler.AnalyzeTrace)

	body := map[string]interface{}{
		"node_name": "HTTP Request",
		"node_type": "http",
	}

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/ai/analyze-trace", testTenantID),
		body, nil)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestAIHandler_AnalyzeTrace_MissingNodeName(t *testing.T) {
	mockAI := new(mocks.MockAIService)
	mockPlatform := new(mocks.MockPlatformClient)

	handler := handlers.NewAIHandler(mockAI, mockPlatform)
	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/ai/analyze-trace", handler.AnalyzeTrace)

	body := map[string]interface{}{
		"error":     "Connection refused",
		"node_type": "http",
	}

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/ai/analyze-trace", testTenantID),
		body, nil)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestAIHandler_AnalyzeTrace_ServiceError(t *testing.T) {
	mockAI := new(mocks.MockAIService)
	mockPlatform := new(mocks.MockPlatformClient)

	mockAI.On("AnalyzeTrace",
		mock.Anything, testTenantID, mock.Anything,
	).Return("", fmt.Errorf("LLM call failed"))

	handler := handlers.NewAIHandler(mockAI, mockPlatform)
	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/ai/analyze-trace", handler.AnalyzeTrace)

	body := dto.AnalyzeTraceRequest{
		NodeName: "HTTP Request",
		NodeType: "http",
		Error:    "Connection refused",
	}

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/ai/analyze-trace", testTenantID),
		body, nil)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)

	mockAI.AssertExpectations(t)
}

func TestAIHandler_AnalyzeTrace_EmptyBody(t *testing.T) {
	mockAI := new(mocks.MockAIService)
	mockPlatform := new(mocks.MockPlatformClient)

	handler := handlers.NewAIHandler(mockAI, mockPlatform)
	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/ai/analyze-trace", handler.AnalyzeTrace)

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/ai/analyze-trace", testTenantID),
		nil, nil)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

// --- SummarizeTrace Tests ---

func TestAIHandler_SummarizeTrace_Success(t *testing.T) {
	mockAI := new(mocks.MockAIService)
	mockPlatform := new(mocks.MockPlatformClient)

	mockAI.On("SummarizeTrace",
		mock.Anything, testTenantID, mock.MatchedBy(func(req ai.SummarizeTraceInput) bool {
			return req.DetailLevel == "short" && len(req.Nodes) == 2
		}),
	).Return("## Trace Summary\n\n**Status:** Completed in 2.3s", nil)

	handler := handlers.NewAIHandler(mockAI, mockPlatform)
	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/ai/summarize-trace", handler.SummarizeTrace)

	body := dto.SummarizeTraceRequest{
		TraceID:     "trace-123",
		DetailLevel: "short",
		Nodes: []map[string]interface{}{
			{"name": "Agent", "type": "agent", "status": "completed", "duration": 1.5},
			{"name": "Tool Call", "type": "tool", "status": "completed", "duration": 0.8},
		},
	}

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/ai/summarize-trace", testTenantID),
		body, nil)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var resp dto.SummarizeTraceResponse
	testutils.ParseJSONResponse(t, w, &resp)
	assert.Contains(t, resp.Summary, "Trace Summary")

	mockAI.AssertExpectations(t)
}

func TestAIHandler_SummarizeTrace_MediumDetail(t *testing.T) {
	mockAI := new(mocks.MockAIService)
	mockPlatform := new(mocks.MockPlatformClient)

	mockAI.On("SummarizeTrace",
		mock.Anything, testTenantID, mock.MatchedBy(func(req ai.SummarizeTraceInput) bool {
			return req.DetailLevel == "medium"
		}),
	).Return("## Detailed Summary\n\nStep-by-step analysis...", nil)

	handler := handlers.NewAIHandler(mockAI, mockPlatform)
	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/ai/summarize-trace", handler.SummarizeTrace)

	body := dto.SummarizeTraceRequest{
		DetailLevel: "medium",
		Nodes:       []map[string]interface{}{{"name": "Agent", "status": "completed"}},
	}

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/ai/summarize-trace", testTenantID),
		body, nil)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	mockAI.AssertExpectations(t)
}

func TestAIHandler_SummarizeTrace_LongDetailLevel(t *testing.T) {
	mockAI := new(mocks.MockAIService)
	mockPlatform := new(mocks.MockPlatformClient)

	mockAI.On("SummarizeTrace",
		mock.Anything, testTenantID, mock.MatchedBy(func(req ai.SummarizeTraceInput) bool {
			return req.DetailLevel == "long"
		}),
	).Return("## Detailed Trace Analysis\n\nComprehensive breakdown...", nil)

	handler := handlers.NewAIHandler(mockAI, mockPlatform)
	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/ai/summarize-trace", handler.SummarizeTrace)

	body := dto.SummarizeTraceRequest{
		DetailLevel: "long",
		Nodes: []map[string]interface{}{
			{
				"name":   "Agent",
				"type":   "agent",
				"status": "completed",
				"data": map[string]interface{}{
					"input":  "user query",
					"output": "agent response",
				},
			},
		},
	}

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/ai/summarize-trace", testTenantID),
		body, nil)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	mockAI.AssertExpectations(t)
}

func TestAIHandler_SummarizeTrace_InvalidDetailLevel(t *testing.T) {
	mockAI := new(mocks.MockAIService)
	mockPlatform := new(mocks.MockPlatformClient)

	handler := handlers.NewAIHandler(mockAI, mockPlatform)
	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/ai/summarize-trace", handler.SummarizeTrace)

	body := map[string]interface{}{
		"detail_level": "invalid",
		"nodes":        []map[string]interface{}{{"name": "Agent"}},
	}

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/ai/summarize-trace", testTenantID),
		body, nil)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestAIHandler_SummarizeTrace_MissingNodes(t *testing.T) {
	mockAI := new(mocks.MockAIService)
	mockPlatform := new(mocks.MockPlatformClient)

	handler := handlers.NewAIHandler(mockAI, mockPlatform)
	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/ai/summarize-trace", handler.SummarizeTrace)

	body := map[string]interface{}{
		"detail_level": "short",
	}

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/ai/summarize-trace", testTenantID),
		body, nil)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

// --- TestModel Tests ---

func TestAIHandler_TestModel_Success(t *testing.T) {
	mockAI := new(mocks.MockAIService)
	mockPlatform := new(mocks.MockPlatformClient)

	mockPlatform.On("GetCredentialSecret",
		mock.Anything, testTenantID, "cred-123", "",
	).Return(`{"api_key": "sk-test-key"}`, nil)

	mockAI.On("TestModel",
		mock.Anything, "OPENAI",
		map[string]interface{}{"model_name": "gpt-4o"},
		map[string]interface{}{"api_key": "sk-test-key"},
	).Return(&ai.TestModelResult{
		Success:        true,
		Message:        "Model responded successfully",
		ResponseTimeMs: 423,
	}, nil)

	handler := handlers.NewAIHandler(mockAI, mockPlatform)
	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/ai/test-model", handler.TestModel)

	body := dto.TestModelRequest{
		Provider:     "OPENAI",
		Config:       map[string]interface{}{"model_name": "gpt-4o"},
		CredentialID: "cred-123",
	}

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/ai/test-model", testTenantID),
		body, nil)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var resp dto.TestModelResponse
	testutils.ParseJSONResponse(t, w, &resp)
	assert.True(t, resp.Success)
	assert.Equal(t, "Model responded successfully", resp.Message)
	assert.Equal(t, int64(423), resp.ResponseTimeMs)

	mockAI.AssertExpectations(t)
	mockPlatform.AssertExpectations(t)
}

func TestAIHandler_TestModel_Failure(t *testing.T) {
	mockAI := new(mocks.MockAIService)
	mockPlatform := new(mocks.MockPlatformClient)

	mockPlatform.On("GetCredentialSecret",
		mock.Anything, testTenantID, "cred-123", "",
	).Return(`{"api_key": "invalid-key"}`, nil)

	mockAI.On("TestModel",
		mock.Anything, "OPENAI",
		map[string]interface{}{"model_name": "gpt-4o"},
		map[string]interface{}{"api_key": "invalid-key"},
	).Return(&ai.TestModelResult{
		Success:        false,
		Message:        "Model test failed: Authentication failed",
		ResponseTimeMs: 150,
	}, nil)

	handler := handlers.NewAIHandler(mockAI, mockPlatform)
	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/ai/test-model", handler.TestModel)

	body := dto.TestModelRequest{
		Provider:     "OPENAI",
		Config:       map[string]interface{}{"model_name": "gpt-4o"},
		CredentialID: "cred-123",
	}

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/ai/test-model", testTenantID),
		body, nil)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var resp dto.TestModelResponse
	testutils.ParseJSONResponse(t, w, &resp)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Message, "Authentication failed")

	mockAI.AssertExpectations(t)
}

func TestAIHandler_TestModel_WithoutCredential(t *testing.T) {
	mockAI := new(mocks.MockAIService)
	mockPlatform := new(mocks.MockPlatformClient)

	mockAI.On("TestModel",
		mock.Anything, "OLLAMA",
		map[string]interface{}{"model_name": "llama3", "base_url": "http://localhost:11434"},
		mock.Anything,
	).Return(&ai.TestModelResult{
		Success:        true,
		Message:        "Model responded successfully",
		ResponseTimeMs: 200,
	}, nil)

	handler := handlers.NewAIHandler(mockAI, mockPlatform)
	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/ai/test-model", handler.TestModel)

	body := dto.TestModelRequest{
		Provider: "OLLAMA",
		Config:   map[string]interface{}{"model_name": "llama3", "base_url": "http://localhost:11434"},
	}

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/ai/test-model", testTenantID),
		body, nil)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var resp dto.TestModelResponse
	testutils.ParseJSONResponse(t, w, &resp)
	assert.True(t, resp.Success)

	mockAI.AssertExpectations(t)
}

func TestAIHandler_TestModel_MissingProvider(t *testing.T) {
	mockAI := new(mocks.MockAIService)
	mockPlatform := new(mocks.MockPlatformClient)

	handler := handlers.NewAIHandler(mockAI, mockPlatform)
	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/ai/test-model", handler.TestModel)

	body := map[string]interface{}{
		"config": map[string]interface{}{"model_name": "gpt-4o"},
	}

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/ai/test-model", testTenantID),
		body, nil)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestAIHandler_TestModel_MissingConfig(t *testing.T) {
	mockAI := new(mocks.MockAIService)
	mockPlatform := new(mocks.MockPlatformClient)

	handler := handlers.NewAIHandler(mockAI, mockPlatform)
	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/ai/test-model", handler.TestModel)

	body := map[string]interface{}{
		"provider": "OPENAI",
	}

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/ai/test-model", testTenantID),
		body, nil)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

func TestAIHandler_TestModel_CredentialFetchError(t *testing.T) {
	mockAI := new(mocks.MockAIService)
	mockPlatform := new(mocks.MockPlatformClient)

	mockPlatform.On("GetCredentialSecret",
		mock.Anything, testTenantID, "bad-cred", "",
	).Return("", fmt.Errorf("not_found: credential not found"))

	handler := handlers.NewAIHandler(mockAI, mockPlatform)
	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/ai/test-model", handler.TestModel)

	body := dto.TestModelRequest{
		Provider:     "OPENAI",
		Config:       map[string]interface{}{"model_name": "gpt-4o"},
		CredentialID: "bad-cred",
	}

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/ai/test-model", testTenantID),
		body, nil)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)

	mockPlatform.AssertExpectations(t)
}

func TestAIHandler_TestModel_AzureOpenAI(t *testing.T) {
	mockAI := new(mocks.MockAIService)
	mockPlatform := new(mocks.MockPlatformClient)

	mockPlatform.On("GetCredentialSecret",
		mock.Anything, testTenantID, "azure-cred", "",
	).Return(`{"api_key": "azure-key-123"}`, nil)

	mockAI.On("TestModel",
		mock.Anything, "AZURE_OPENAI",
		map[string]interface{}{
			"endpoint":        "https://my-resource.openai.azure.com/",
			"api_version":     "2024-12-01-preview",
			"deployment_name": "gpt-4o",
		},
		map[string]interface{}{"api_key": "azure-key-123"},
	).Return(&ai.TestModelResult{
		Success:        true,
		Message:        "Model responded successfully",
		ResponseTimeMs: 500,
	}, nil)

	handler := handlers.NewAIHandler(mockAI, mockPlatform)
	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/ai/test-model", handler.TestModel)

	body := dto.TestModelRequest{
		Provider: "AZURE_OPENAI",
		Config: map[string]interface{}{
			"endpoint":        "https://my-resource.openai.azure.com/",
			"api_version":     "2024-12-01-preview",
			"deployment_name": "gpt-4o",
		},
		CredentialID: "azure-cred",
	}

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/ai/test-model", testTenantID),
		body, nil)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var resp dto.TestModelResponse
	testutils.ParseJSONResponse(t, w, &resp)
	assert.True(t, resp.Success)

	mockAI.AssertExpectations(t)
	mockPlatform.AssertExpectations(t)
}

func TestAIHandler_TestModel_EmptyBody(t *testing.T) {
	mockAI := new(mocks.MockAIService)
	mockPlatform := new(mocks.MockPlatformClient)

	handler := handlers.NewAIHandler(mockAI, mockPlatform)
	router := testutils.SetupTestRouter()
	router.POST("/tenants/:tenantId/ai/test-model", handler.TestModel)

	w := testutils.PerformRequest(router, "POST",
		fmt.Sprintf("/tenants/%s/ai/test-model", testTenantID),
		nil, nil)

	testutils.AssertStatusCode(t, http.StatusBadRequest, w)
}

// --- GetCapabilities Tests ---

func TestAIHandler_GetCapabilities_AllEnabled(t *testing.T) {
	mockAI := new(mocks.MockAIService)
	mockPlatform := new(mocks.MockPlatformClient)

	mockAI.On("GetCapabilities",
		mock.Anything, testTenantID,
	).Return(&ai.Capabilities{
		TitleGeneration:       true,
		DescriptionGeneration: true,
		TraceAnalysis:         true,
		Summarization:         true,
	}, nil)

	handler := handlers.NewAIHandler(mockAI, mockPlatform)
	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/ai/capabilities", handler.GetCapabilities)

	w := testutils.PerformRequest(router, "GET",
		fmt.Sprintf("/tenants/%s/ai/capabilities", testTenantID),
		nil, nil)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var resp dto.AICapabilitiesResponse
	testutils.ParseJSONResponse(t, w, &resp)
	assert.True(t, resp.TitleGeneration)
	assert.True(t, resp.DescriptionGeneration)
	assert.True(t, resp.TraceAnalysis)
	assert.True(t, resp.Summarization)

	mockAI.AssertExpectations(t)
}

func TestAIHandler_GetCapabilities_NoneEnabled(t *testing.T) {
	mockAI := new(mocks.MockAIService)
	mockPlatform := new(mocks.MockPlatformClient)

	mockAI.On("GetCapabilities",
		mock.Anything, testTenantID,
	).Return(&ai.Capabilities{
		TitleGeneration:       false,
		DescriptionGeneration: false,
		TraceAnalysis:         false,
		Summarization:         false,
	}, nil)

	handler := handlers.NewAIHandler(mockAI, mockPlatform)
	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/ai/capabilities", handler.GetCapabilities)

	w := testutils.PerformRequest(router, "GET",
		fmt.Sprintf("/tenants/%s/ai/capabilities", testTenantID),
		nil, nil)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var resp dto.AICapabilitiesResponse
	testutils.ParseJSONResponse(t, w, &resp)
	assert.False(t, resp.TitleGeneration)
	assert.False(t, resp.DescriptionGeneration)
	assert.False(t, resp.TraceAnalysis)
	assert.False(t, resp.Summarization)

	mockAI.AssertExpectations(t)
}

func TestAIHandler_GetCapabilities_PartialEnabled(t *testing.T) {
	mockAI := new(mocks.MockAIService)
	mockPlatform := new(mocks.MockPlatformClient)

	mockAI.On("GetCapabilities",
		mock.Anything, testTenantID,
	).Return(&ai.Capabilities{
		TitleGeneration:       true,
		DescriptionGeneration: true,
		TraceAnalysis:         false,
		Summarization:         false,
	}, nil)

	handler := handlers.NewAIHandler(mockAI, mockPlatform)
	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/ai/capabilities", handler.GetCapabilities)

	w := testutils.PerformRequest(router, "GET",
		fmt.Sprintf("/tenants/%s/ai/capabilities", testTenantID),
		nil, nil)

	testutils.AssertStatusCode(t, http.StatusOK, w)

	var resp dto.AICapabilitiesResponse
	testutils.ParseJSONResponse(t, w, &resp)
	assert.True(t, resp.TitleGeneration)
	assert.True(t, resp.DescriptionGeneration)
	assert.False(t, resp.TraceAnalysis)
	assert.False(t, resp.Summarization)

	mockAI.AssertExpectations(t)
}

func TestAIHandler_GetCapabilities_ServiceError(t *testing.T) {
	mockAI := new(mocks.MockAIService)
	mockPlatform := new(mocks.MockPlatformClient)

	mockAI.On("GetCapabilities",
		mock.Anything, testTenantID,
	).Return(nil, fmt.Errorf("platform service unavailable"))

	handler := handlers.NewAIHandler(mockAI, mockPlatform)
	router := testutils.SetupTestRouter()
	router.GET("/tenants/:tenantId/ai/capabilities", handler.GetCapabilities)

	w := testutils.PerformRequest(router, "GET",
		fmt.Sprintf("/tenants/%s/ai/capabilities", testTenantID),
		nil, nil)

	testutils.AssertStatusCode(t, http.StatusInternalServerError, w)

	mockAI.AssertExpectations(t)
}
