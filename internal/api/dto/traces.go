// Package dto provides Data Transfer Objects for traces API requests and responses.
package dto

import (
	"time"

	"github.com/unifiedui/agent-service/internal/domain/models"
)

// --- Request DTOs ---

// NodeDataIORequest represents input/output data for a node in requests.
type NodeDataIORequest struct {
	Text      string                 `json:"text,omitempty"`
	ExtraData map[string]interface{} `json:"extraData,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NodeDataRequest represents node data in requests.
type NodeDataRequest struct {
	Input  *NodeDataIORequest `json:"input,omitempty"`
	Output *NodeDataIORequest `json:"output,omitempty"`
}

// TraceNodeRequest represents a trace node in API requests.
type TraceNodeRequest struct {
	ID          string                 `json:"id" binding:"required"`
	Name        string                 `json:"name" binding:"required"`
	Type        string                 `json:"type" binding:"required"`
	ReferenceID string                 `json:"referenceId,omitempty"`
	StartAt     *time.Time             `json:"startAt,omitempty"`
	EndAt       *time.Time             `json:"endAt,omitempty"`
	Duration    float64                `json:"duration,omitempty"`
	Status      string                 `json:"status" binding:"required"`
	Logs        []interface{}          `json:"logs,omitempty"`
	Data        *NodeDataRequest       `json:"data,omitempty"`
	Nodes       []TraceNodeRequest     `json:"nodes,omitempty"` // Sub-nodes
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// CreateTraceRequest represents the request body for creating a trace.
type CreateTraceRequest struct {
	// ID is optional; if not provided, one will be generated.
	ID string `json:"id,omitempty"`

	// Context fields - EITHER (applicationId + conversationId) OR autonomousAgentId
	ApplicationID     string `json:"applicationId,omitempty"`
	ConversationID    string `json:"conversationId,omitempty"`
	AutonomousAgentID string `json:"autonomousAgentId,omitempty"`

	// Reference fields for external system linkage.
	ReferenceID       string                 `json:"referenceId,omitempty"`
	ReferenceName     string                 `json:"referenceName,omitempty"`
	ReferenceMetadata map[string]interface{} `json:"referenceMetadata,omitempty"`

	// Initial logs (optional).
	Logs []interface{} `json:"logs,omitempty"`

	// Initial nodes (optional).
	Nodes []TraceNodeRequest `json:"nodes,omitempty"`
}

// AddNodesRequest represents the request body for adding nodes to a trace.
type AddNodesRequest struct {
	Nodes []TraceNodeRequest `json:"nodes" binding:"required,min=1"`
}

// AddLogsRequest represents the request body for adding logs to a trace.
type AddLogsRequest struct {
	Logs []interface{} `json:"logs" binding:"required,min=1"`
}

// RefreshTraceRequest represents the request body for refreshing (replacing) a trace.
type RefreshTraceRequest struct {
	// Reference fields for external system linkage.
	ReferenceID       string                 `json:"referenceId,omitempty"`
	ReferenceName     string                 `json:"referenceName,omitempty"`
	ReferenceMetadata map[string]interface{} `json:"referenceMetadata,omitempty"`

	// Logs to set.
	Logs []interface{} `json:"logs,omitempty"`

	// Nodes to set.
	Nodes []TraceNodeRequest `json:"nodes,omitempty"`
}

// --- Response DTOs ---

// NodeDataIOResponse represents input/output data for a node in responses.
type NodeDataIOResponse struct {
	Text      string                 `json:"text,omitempty"`
	ExtraData map[string]interface{} `json:"extraData,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NodeDataResponse represents node data in responses.
type NodeDataResponse struct {
	Input  *NodeDataIOResponse `json:"input,omitempty"`
	Output *NodeDataIOResponse `json:"output,omitempty"`
}

// TraceNodeResponse represents a trace node in API responses.
type TraceNodeResponse struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	ReferenceID string                 `json:"referenceId,omitempty"`
	StartAt     *time.Time             `json:"startAt,omitempty"`
	EndAt       *time.Time             `json:"endAt,omitempty"`
	Duration    float64                `json:"duration,omitempty"`
	Status      string                 `json:"status"`
	Logs        []string               `json:"logs,omitempty"`
	Data        *NodeDataResponse      `json:"data,omitempty"`
	Nodes       []TraceNodeResponse    `json:"nodes,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
	CreatedBy   string                 `json:"createdBy,omitempty"`
	UpdatedBy   string                 `json:"updatedBy,omitempty"`
}

// TraceResponse represents a trace in API responses.
type TraceResponse struct {
	ID                string                 `json:"id"`
	TenantID          string                 `json:"tenantId"`
	ApplicationID     string                 `json:"applicationId,omitempty"`
	ConversationID    string                 `json:"conversationId,omitempty"`
	AutonomousAgentID string                 `json:"autonomousAgentId,omitempty"`
	ContextType       string                 `json:"contextType"`
	ReferenceID       string                 `json:"referenceId,omitempty"`
	ReferenceName     string                 `json:"referenceName,omitempty"`
	ReferenceMetadata map[string]interface{} `json:"referenceMetadata,omitempty"`
	Logs              []string               `json:"logs,omitempty"`
	Nodes             []TraceNodeResponse    `json:"nodes,omitempty"`
	CreatedAt         time.Time              `json:"createdAt"`
	UpdatedAt         time.Time              `json:"updatedAt"`
	CreatedBy         string                 `json:"createdBy,omitempty"`
	UpdatedBy         string                 `json:"updatedBy,omitempty"`
}

// ListTracesResponse represents the response for listing traces.
type ListTracesResponse struct {
	Traces []*TraceResponse `json:"traces"`
}

// CreateTraceResponse represents the response for creating a trace.
type CreateTraceResponse struct {
	ID string `json:"id"`
}

// --- Transformation Functions ---

// ToTraceNode converts a TraceNodeRequest to a models.TraceNode.
func (r *TraceNodeRequest) ToTraceNode(createdBy string) models.TraceNode {
	now := time.Now().UTC()
	node := models.TraceNode{
		ID:          r.ID,
		Name:        r.Name,
		Type:        models.NodeType(r.Type),
		ReferenceID: r.ReferenceID,
		StartAt:     r.StartAt,
		EndAt:       r.EndAt,
		Duration:    r.Duration,
		Status:      models.NodeStatus(r.Status),
		Logs:        models.ConvertLogsToStrings(r.Logs),
		Metadata:    r.Metadata,
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   createdBy,
		UpdatedBy:   createdBy,
	}

	if r.Data != nil {
		node.Data = &models.NodeData{}
		if r.Data.Input != nil {
			node.Data.Input = &models.NodeDataIO{
				Text:      r.Data.Input.Text,
				ExtraData: r.Data.Input.ExtraData,
				Metadata:  r.Data.Input.Metadata,
			}
		}
		if r.Data.Output != nil {
			node.Data.Output = &models.NodeDataIO{
				Text:      r.Data.Output.Text,
				ExtraData: r.Data.Output.ExtraData,
				Metadata:  r.Data.Output.Metadata,
			}
		}
	}

	// Convert sub-nodes recursively
	if len(r.Nodes) > 0 {
		node.Nodes = make([]models.TraceNode, len(r.Nodes))
		for i, subNode := range r.Nodes {
			node.Nodes[i] = subNode.ToTraceNode(createdBy)
		}
	}

	return node
}

// TraceNodeToResponse converts a models.TraceNode to a TraceNodeResponse.
func TraceNodeToResponse(node models.TraceNode) TraceNodeResponse {
	resp := TraceNodeResponse{
		ID:          node.ID,
		Name:        node.Name,
		Type:        string(node.Type),
		ReferenceID: node.ReferenceID,
		StartAt:     node.StartAt,
		EndAt:       node.EndAt,
		Duration:    node.Duration,
		Status:      string(node.Status),
		Logs:        node.Logs,
		Metadata:    node.Metadata,
		CreatedAt:   node.CreatedAt,
		UpdatedAt:   node.UpdatedAt,
		CreatedBy:   node.CreatedBy,
		UpdatedBy:   node.UpdatedBy,
	}

	if node.Data != nil {
		resp.Data = &NodeDataResponse{}
		if node.Data.Input != nil {
			resp.Data.Input = &NodeDataIOResponse{
				Text:      node.Data.Input.Text,
				ExtraData: node.Data.Input.ExtraData,
				Metadata:  node.Data.Input.Metadata,
			}
		}
		if node.Data.Output != nil {
			resp.Data.Output = &NodeDataIOResponse{
				Text:      node.Data.Output.Text,
				ExtraData: node.Data.Output.ExtraData,
				Metadata:  node.Data.Output.Metadata,
			}
		}
	}

	// Convert sub-nodes recursively
	if len(node.Nodes) > 0 {
		resp.Nodes = make([]TraceNodeResponse, len(node.Nodes))
		for i, subNode := range node.Nodes {
			resp.Nodes[i] = TraceNodeToResponse(subNode)
		}
	}

	return resp
}

// TraceToResponse converts a models.Trace to a TraceResponse.
func TraceToResponse(trace *models.Trace) *TraceResponse {
	if trace == nil {
		return nil
	}

	resp := &TraceResponse{
		ID:                trace.ID,
		TenantID:          trace.TenantID,
		ApplicationID:     trace.ApplicationID,
		ConversationID:    trace.ConversationID,
		AutonomousAgentID: trace.AutonomousAgentID,
		ContextType:       string(trace.ContextType),
		ReferenceID:       trace.ReferenceID,
		ReferenceName:     trace.ReferenceName,
		ReferenceMetadata: trace.ReferenceMetadata,
		Logs:              trace.Logs,
		CreatedAt:         trace.CreatedAt,
		UpdatedAt:         trace.UpdatedAt,
		CreatedBy:         trace.CreatedBy,
		UpdatedBy:         trace.UpdatedBy,
	}

	// Convert nodes
	if len(trace.Nodes) > 0 {
		resp.Nodes = make([]TraceNodeResponse, len(trace.Nodes))
		for i, node := range trace.Nodes {
			resp.Nodes[i] = TraceNodeToResponse(node)
		}
	}

	return resp
}

// TracesToResponse converts a slice of models.Trace to a slice of TraceResponse.
func TracesToResponse(traces []*models.Trace) []*TraceResponse {
	if traces == nil {
		return []*TraceResponse{}
	}

	responses := make([]*TraceResponse, len(traces))
	for i, trace := range traces {
		responses[i] = TraceToResponse(trace)
	}
	return responses
}

// ConvertNodesToModel converts request nodes to model nodes.
func ConvertNodesToModel(nodes []TraceNodeRequest, createdBy string) []models.TraceNode {
	if nodes == nil {
		return []models.TraceNode{}
	}

	result := make([]models.TraceNode, len(nodes))
	for i, node := range nodes {
		result[i] = node.ToTraceNode(createdBy)
	}
	return result
}

// --- Import DTOs ---

// ImportTraceResponse represents the response for importing traces.
type ImportTraceResponse struct {
	Message string `json:"message"`
	TraceID string `json:"traceId,omitempty"`
}
