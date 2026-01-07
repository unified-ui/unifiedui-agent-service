// Package models contains domain models for the UnifiedUI Agent Service.
package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// NodeStatus represents the status of a trace node.
type NodeStatus string

const (
	// NodeStatusPending indicates the node is pending.
	NodeStatusPending NodeStatus = "pending"
	// NodeStatusRunning indicates the node is running.
	NodeStatusRunning NodeStatus = "running"
	// NodeStatusCompleted indicates the node completed successfully.
	NodeStatusCompleted NodeStatus = "completed"
	// NodeStatusFailed indicates the node failed.
	NodeStatusFailed NodeStatus = "failed"
	// NodeStatusSkipped indicates the node was skipped.
	NodeStatusSkipped NodeStatus = "skipped"
	// NodeStatusCancelled indicates the node was cancelled.
	NodeStatusCancelled NodeStatus = "cancelled"
)

// NodeType represents the type of a trace node.
type NodeType string

const (
	// NodeTypeAgent represents an agent node.
	NodeTypeAgent NodeType = "agent"
	// NodeTypeTool represents a tool node.
	NodeTypeTool NodeType = "tool"
	// NodeTypeLLM represents an LLM node.
	NodeTypeLLM NodeType = "llm"
	// NodeTypeChain represents a chain node.
	NodeTypeChain NodeType = "chain"
	// NodeTypeRetriever represents a retriever node.
	NodeTypeRetriever NodeType = "retriever"
	// NodeTypeWorkflow represents a workflow node.
	NodeTypeWorkflow NodeType = "workflow"
	// NodeTypeFunction represents a function node.
	NodeTypeFunction NodeType = "function"
	// NodeTypeHTTP represents an HTTP node.
	NodeTypeHTTP NodeType = "http"
	// NodeTypeCode represents a code execution node.
	NodeTypeCode NodeType = "code"
	// NodeTypeConditional represents a conditional node.
	NodeTypeConditional NodeType = "conditional"
	// NodeTypeLoop represents a loop node.
	NodeTypeLoop NodeType = "loop"
	// NodeTypeCustom represents a custom node type.
	NodeTypeCustom NodeType = "custom"
)

// TraceContextType represents the context type of a trace.
type TraceContextType string

const (
	// TraceContextConversation indicates trace is linked to a conversation.
	TraceContextConversation TraceContextType = "conversation"
	// TraceContextAutonomousAgent indicates trace is linked to an autonomous agent.
	TraceContextAutonomousAgent TraceContextType = "autonomous_agent"
)

// NodeDataIO represents input or output data for a node.
type NodeDataIO struct {
	Text      string                 `json:"text,omitempty" bson:"text,omitempty"`
	ExtraData map[string]interface{} `json:"extraData,omitempty" bson:"extraData,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty" bson:"metadata,omitempty"`
}

// NodeData represents the data (input/output) of a node.
type NodeData struct {
	Input  *NodeDataIO `json:"input,omitempty" bson:"input,omitempty"`
	Output *NodeDataIO `json:"output,omitempty" bson:"output,omitempty"`
}

// TraceNode represents a single node in the trace tree.
// Nodes can be hierarchical with sub-nodes.
type TraceNode struct {
	ID          string                 `json:"id" bson:"id"`
	Name        string                 `json:"name" bson:"name"`
	Type        NodeType               `json:"type" bson:"type"`
	ReferenceID string                 `json:"referenceId,omitempty" bson:"referenceId,omitempty"`
	StartAt     *time.Time             `json:"startAt,omitempty" bson:"startAt,omitempty"`
	EndAt       *time.Time             `json:"endAt,omitempty" bson:"endAt,omitempty"`
	Duration    float64                `json:"duration,omitempty" bson:"duration,omitempty"` // Duration in seconds
	Status      NodeStatus             `json:"status" bson:"status"`
	Logs        []string               `json:"logs,omitempty" bson:"logs,omitempty"`
	Data        *NodeData              `json:"data,omitempty" bson:"data,omitempty"`
	Nodes       []TraceNode            `json:"nodes,omitempty" bson:"nodes,omitempty"` // Sub-nodes (hierarchical)
	Metadata    map[string]interface{} `json:"metadata,omitempty" bson:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"createdAt" bson:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt" bson:"updatedAt"`
	CreatedBy   string                 `json:"createdBy,omitempty" bson:"createdBy,omitempty"`
	UpdatedBy   string                 `json:"updatedBy,omitempty" bson:"updatedBy,omitempty"`
}

// Trace represents a complete trace for a workflow execution.
// A trace contains hierarchical nodes representing the execution flow.
type Trace struct {
	// ID is the unique identifier for this trace.
	ID string `json:"id" bson:"_id"`

	// TenantID is required for tenant isolation.
	TenantID string `json:"tenantId" bson:"tenantId"`

	// Context fields - either (ApplicationID + ConversationID) OR AutonomousAgentID
	ApplicationID     string `json:"applicationId,omitempty" bson:"applicationId,omitempty"`
	ConversationID    string `json:"conversationId,omitempty" bson:"conversationId,omitempty"`
	AutonomousAgentID string `json:"autonomousAgentId,omitempty" bson:"autonomousAgentId,omitempty"`

	// ContextType indicates whether this trace is for a conversation or autonomous agent.
	ContextType TraceContextType `json:"contextType" bson:"contextType"`

	// Reference fields for external system linkage.
	ReferenceID       string                 `json:"referenceId,omitempty" bson:"referenceId,omitempty"`
	ReferenceName     string                 `json:"referenceName,omitempty" bson:"referenceName,omitempty"`
	ReferenceMetadata map[string]interface{} `json:"referenceMetadata,omitempty" bson:"referenceMetadata,omitempty"`

	// Logs at the trace level. Each log entry is stored as a JSON string.
	Logs []string `json:"logs,omitempty" bson:"logs,omitempty"`

	// Nodes contains the hierarchical execution tree.
	Nodes []TraceNode `json:"nodes,omitempty" bson:"nodes,omitempty"`

	// Audit fields
	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
	CreatedBy string    `json:"createdBy,omitempty" bson:"createdBy,omitempty"`
	UpdatedBy string    `json:"updatedBy,omitempty" bson:"updatedBy,omitempty"`
}

// NewConversationTrace creates a new trace for a conversation context.
func NewConversationTrace(tenantID, applicationID, conversationID, createdBy string) *Trace {
	now := time.Now().UTC()
	return &Trace{
		TenantID:       tenantID,
		ApplicationID:  applicationID,
		ConversationID: conversationID,
		ContextType:    TraceContextConversation,
		Nodes:          []TraceNode{},
		Logs:           []string{},
		CreatedAt:      now,
		UpdatedAt:      now,
		CreatedBy:      createdBy,
		UpdatedBy:      createdBy,
	}
}

// NewAutonomousAgentTrace creates a new trace for an autonomous agent context.
func NewAutonomousAgentTrace(tenantID, autonomousAgentID, createdBy string) *Trace {
	now := time.Now().UTC()
	return &Trace{
		TenantID:          tenantID,
		AutonomousAgentID: autonomousAgentID,
		ContextType:       TraceContextAutonomousAgent,
		Nodes:             []TraceNode{},
		Logs:              []string{},
		CreatedAt:         now,
		UpdatedAt:         now,
		CreatedBy:         createdBy,
		UpdatedBy:         createdBy,
	}
}

// NewTraceNode creates a new trace node.
func NewTraceNode(id, name string, nodeType NodeType, createdBy string) TraceNode {
	now := time.Now().UTC()
	return TraceNode{
		ID:        id,
		Name:      name,
		Type:      nodeType,
		Status:    NodeStatusPending,
		Nodes:     []TraceNode{},
		Logs:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: createdBy,
		UpdatedBy: createdBy,
	}
}

// AddNode adds a node to the trace.
func (t *Trace) AddNode(node TraceNode) {
	t.Nodes = append(t.Nodes, node)
	t.UpdatedAt = time.Now().UTC()
}

// AddNodes adds multiple nodes to the trace.
func (t *Trace) AddNodes(nodes []TraceNode) {
	t.Nodes = append(t.Nodes, nodes...)
	t.UpdatedAt = time.Now().UTC()
}

// AddLog adds a log entry to the trace.
func (t *Trace) AddLog(log string) {
	t.Logs = append(t.Logs, log)
	t.UpdatedAt = time.Now().UTC()
}

// AddLogs adds multiple log entries to the trace.
func (t *Trace) AddLogs(logs []string) {
	t.Logs = append(t.Logs, logs...)
	t.UpdatedAt = time.Now().UTC()
}

// SetUpdatedBy updates the updatedBy and updatedAt fields.
func (t *Trace) SetUpdatedBy(updatedBy string) {
	t.UpdatedBy = updatedBy
	t.UpdatedAt = time.Now().UTC()
}

// IsConversationContext returns true if the trace is for a conversation context.
func (t *Trace) IsConversationContext() bool {
	return t.ContextType == TraceContextConversation
}

// IsAutonomousAgentContext returns true if the trace is for an autonomous agent context.
func (t *Trace) IsAutonomousAgentContext() bool {
	return t.ContextType == TraceContextAutonomousAgent
}

// ValidateContext validates that the trace has valid context fields.
func (t *Trace) ValidateContext() bool {
	if t.ContextType == TraceContextConversation {
		return t.ApplicationID != "" && t.ConversationID != "" && t.AutonomousAgentID == ""
	}
	if t.ContextType == TraceContextAutonomousAgent {
		return t.AutonomousAgentID != "" && t.ApplicationID == "" && t.ConversationID == ""
	}
	return false
}

// Validate validates the trace and returns an error if invalid.
func (t *Trace) Validate() error {
	if t.TenantID == "" {
		return fmt.Errorf("tenantId is required")
	}

	// Check for mixed context
	hasConversationContext := t.ApplicationID != "" || t.ConversationID != ""
	hasAutonomousAgentContext := t.AutonomousAgentID != ""

	if hasConversationContext && hasAutonomousAgentContext {
		return fmt.Errorf("cannot have both conversation and autonomous agent context")
	}

	// Validate based on context type
	if t.ContextType == TraceContextConversation {
		if t.ApplicationID == "" {
			return fmt.Errorf("applicationId is required for conversation context")
		}
		if t.ConversationID == "" {
			return fmt.Errorf("conversationId is required for conversation context")
		}
	} else if t.ContextType == TraceContextAutonomousAgent {
		if t.AutonomousAgentID == "" {
			return fmt.Errorf("autonomousAgentId is required for autonomous agent context")
		}
	} else {
		return fmt.Errorf("invalid context type: %s", t.ContextType)
	}

	// Validate nodes
	for _, node := range t.Nodes {
		if err := node.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// IsValid checks if a NodeStatus is valid.
func (s NodeStatus) IsValid() bool {
	switch s {
	case NodeStatusPending, NodeStatusRunning, NodeStatusCompleted,
		NodeStatusFailed, NodeStatusSkipped, NodeStatusCancelled:
		return true
	}
	return false
}

// IsValid checks if a NodeType is valid.
func (t NodeType) IsValid() bool {
	switch t {
	case NodeTypeAgent, NodeTypeTool, NodeTypeLLM, NodeTypeChain,
		NodeTypeRetriever, NodeTypeWorkflow, NodeTypeFunction,
		NodeTypeHTTP, NodeTypeCode, NodeTypeConditional,
		NodeTypeLoop, NodeTypeCustom:
		return true
	}
	return false
}

// Validate validates a TraceNode and returns an error if invalid.
func (n *TraceNode) Validate() error {
	if n.ID == "" {
		return fmt.Errorf("node id is required")
	}
	if n.Name == "" {
		return fmt.Errorf("node name is required")
	}
	if !n.Type.IsValid() {
		return fmt.Errorf("invalid node type: %s", n.Type)
	}
	if !n.Status.IsValid() {
		return fmt.Errorf("invalid node status: %s", n.Status)
	}

	// Validate sub-nodes recursively
	for _, subNode := range n.Nodes {
		if err := subNode.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// ConvertLogsToStrings converts a slice of interface{} to a slice of strings.
// Each element is converted to a JSON string. If the element is already a string,
// it's kept as-is. Otherwise, it's marshaled to JSON.
func ConvertLogsToStrings(logs []interface{}) []string {
	if logs == nil {
		return []string{}
	}

	result := make([]string, len(logs))
	for i, log := range logs {
		switch v := log.(type) {
		case string:
			result[i] = v
		default:
			// Marshal to JSON string
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				// Fallback to fmt.Sprintf if JSON marshaling fails
				result[i] = fmt.Sprintf("%v", v)
			} else {
				result[i] = string(jsonBytes)
			}
		}
	}
	return result
}
