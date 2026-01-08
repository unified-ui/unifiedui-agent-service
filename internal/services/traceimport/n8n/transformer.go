// Package n8n provides N8N trace import functionality.
package n8n

import (
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/unifiedui/agent-service/internal/domain/models"
)

// Transformer transforms N8N execution data into TraceNodes.
type Transformer struct{}

// NewTransformer creates a new N8N transformer.
func NewTransformer() *Transformer {
	return &Transformer{}
}

// TransformExecution converts an N8N execution response into a list of TraceNodes.
// Each node in the execution's runData becomes a TraceNode.
// Nodes are ordered chronologically by start time.
func (t *Transformer) TransformExecution(execution *ExecutionResponse, createdBy string) []models.TraceNode {
	if execution == nil || execution.Data == nil || execution.Data.ResultData == nil {
		return []models.TraceNode{}
	}

	runData := execution.Data.ResultData.RunData
	if len(runData) == 0 {
		return []models.TraceNode{}
	}

	// Build workflow node map for type lookup
	workflowNodeMap := t.buildWorkflowNodeMap(execution.WorkflowData)

	// Transform each node's executions into TraceNodes
	var allNodes []models.TraceNode

	for nodeName, nodeExecutions := range runData {
		// Get node type from workflow data
		nodeType := ""
		if wfNode, exists := workflowNodeMap[nodeName]; exists {
			nodeType = wfNode.Type
		}

		for runIndex, nodeExec := range nodeExecutions {
			traceNode := t.transformNodeExecution(nodeName, nodeType, runIndex, &nodeExec, createdBy)
			allNodes = append(allNodes, traceNode)
		}
	}

	// Sort nodes by start time (chronological order)
	sort.Slice(allNodes, func(i, j int) bool {
		if allNodes[i].StartAt == nil {
			return true
		}
		if allNodes[j].StartAt == nil {
			return false
		}
		return allNodes[i].StartAt.Before(*allNodes[j].StartAt)
	})

	return allNodes
}

// Transform implements the generic interface for transforming items.
func (t *Transformer) Transform(items interface{}, createdBy string) []models.TraceNode {
	if execution, ok := items.(*ExecutionResponse); ok {
		return t.TransformExecution(execution, createdBy)
	}
	return []models.TraceNode{}
}

// buildWorkflowNodeMap creates a map from node name to workflow node definition.
func (t *Transformer) buildWorkflowNodeMap(workflowData *WorkflowData) map[string]WorkflowNode {
	nodeMap := make(map[string]WorkflowNode)
	if workflowData == nil {
		return nodeMap
	}

	for _, node := range workflowData.Nodes {
		nodeMap[node.Name] = node
	}

	return nodeMap
}

// transformNodeExecution converts a single N8N node execution to a TraceNode.
func (t *Transformer) transformNodeExecution(
	nodeName string,
	nodeType string,
	runIndex int,
	nodeExec *NodeExecution,
	createdBy string,
) models.TraceNode {
	now := time.Now().UTC()

	// Convert N8N node type to our NodeType
	traceNodeType := t.mapNodeType(nodeType)

	// Convert execution status
	status := t.mapNodeStatus(nodeExec.ExecutionStatus, nodeExec.Error)

	// Parse start time
	var startAt *time.Time
	if nodeExec.StartTime > 0 {
		startTime := time.UnixMilli(nodeExec.StartTime).UTC()
		startAt = &startTime
	}

	// Calculate end time
	var endAt *time.Time
	if startAt != nil && nodeExec.ExecutionTime > 0 {
		endTime := startAt.Add(time.Duration(nodeExec.ExecutionTime) * time.Millisecond)
		endAt = &endTime
	}

	// Calculate duration in seconds
	duration := float64(nodeExec.ExecutionTime) / 1000.0

	// Build node data (input/output)
	nodeData := t.buildNodeData(nodeExec, nodeType)

	// Build metadata
	metadata := t.buildNodeMetadata(nodeExec, nodeType, runIndex)

	// Create unique ID
	nodeID := "n8n_node_" + uuid.New().String()

	return models.TraceNode{
		ID:          nodeID,
		ReferenceID: nodeName,
		Name:        nodeName,
		Type:        traceNodeType,
		Status:      status,
		StartAt:     startAt,
		EndAt:       endAt,
		Duration:    duration,
		Data:        nodeData,
		Metadata:    metadata,
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   createdBy,
		UpdatedBy:   createdBy,
	}
}

// mapNodeType converts N8N node type to our internal NodeType.
func (t *Transformer) mapNodeType(n8nType string) models.NodeType {
	// Check for trigger nodes
	if strings.Contains(n8nType, "Trigger") || strings.Contains(n8nType, "trigger") {
		return models.NodeTypeWorkflow
	}

	// Check for agent nodes (AI Agents)
	if strings.Contains(n8nType, "agent") || strings.Contains(n8nType, "Agent") {
		return models.NodeTypeAgent
	}

	// Check for LLM nodes
	if strings.Contains(n8nType, "lmChat") || strings.Contains(n8nType, "LmChat") ||
		strings.Contains(n8nType, "openAi") || strings.Contains(n8nType, "OpenAi") ||
		strings.Contains(n8nType, "anthropic") || strings.Contains(n8nType, "Anthropic") {
		return models.NodeTypeLLM
	}

	// Check for HTTP request nodes
	if strings.Contains(n8nType, "httpRequest") || strings.Contains(n8nType, "HttpRequest") {
		return models.NodeTypeHTTP
	}

	// Check for code/function nodes
	if strings.Contains(n8nType, "code") || strings.Contains(n8nType, "Code") ||
		strings.Contains(n8nType, "function") || strings.Contains(n8nType, "Function") {
		return models.NodeTypeCode
	}

	// Check for conditional nodes
	if strings.Contains(n8nType, "switch") || strings.Contains(n8nType, "Switch") ||
		strings.Contains(n8nType, "if") || strings.Contains(n8nType, "If") {
		return models.NodeTypeConditional
	}

	// Check for merge nodes
	if strings.Contains(n8nType, "merge") || strings.Contains(n8nType, "Merge") {
		return models.NodeTypeWorkflow
	}

	// Check for database nodes
	if strings.Contains(n8nType, "postgres") || strings.Contains(n8nType, "Postgres") ||
		strings.Contains(n8nType, "mongo") || strings.Contains(n8nType, "Mongo") ||
		strings.Contains(n8nType, "mysql") || strings.Contains(n8nType, "MySql") ||
		strings.Contains(n8nType, "redis") || strings.Contains(n8nType, "Redis") {
		return models.NodeTypeTool
	}

	// Check for tool nodes
	if strings.Contains(n8nType, "tool") || strings.Contains(n8nType, "Tool") {
		return models.NodeTypeTool
	}

	// Default to custom type
	return models.NodeTypeCustom
}

// mapNodeStatus converts N8N execution status to our internal NodeStatus.
func (t *Transformer) mapNodeStatus(status NodeExecutionStatus, nodeError *NodeExecutionError) models.NodeStatus {
	// If there's an error, mark as failed
	if nodeError != nil {
		return models.NodeStatusFailed
	}

	switch status {
	case NodeExecutionStatusSuccess:
		return models.NodeStatusCompleted
	case NodeExecutionStatusError:
		return models.NodeStatusFailed
	default:
		// Unknown status, assume completed if we got here
		return models.NodeStatusCompleted
	}
}

// buildNodeData constructs the NodeData structure from node execution.
func (t *Transformer) buildNodeData(nodeExec *NodeExecution, nodeType string) *models.NodeData {
	if nodeExec == nil {
		return nil
	}

	nodeData := &models.NodeData{}

	// Extract input data
	input := t.extractInputData(nodeExec, nodeType)
	if input != nil {
		nodeData.Input = input
	}

	// Extract output data
	output := t.extractOutputData(nodeExec, nodeType)
	if output != nil {
		nodeData.Output = output
	}

	// Only return if we have data
	if nodeData.Input == nil && nodeData.Output == nil {
		return nil
	}

	return nodeData
}

// extractInputData extracts input data from node execution.
func (t *Transformer) extractInputData(nodeExec *NodeExecution, nodeType string) *models.NodeDataIO {
	// Check for input override
	if len(nodeExec.InputOverride) > 0 {
		return &models.NodeDataIO{
			ExtraData: nodeExec.InputOverride,
		}
	}

	// For chat triggers, try to extract chat input
	if strings.Contains(nodeType, "chatTrigger") || strings.Contains(nodeType, "ChatTrigger") {
		if len(nodeExec.Data.Main) > 0 && len(nodeExec.Data.Main[0]) > 0 {
			firstItem := nodeExec.Data.Main[0][0]
			if chatInput, ok := firstItem.JSON["chatInput"].(string); ok {
				return &models.NodeDataIO{
					Text: chatInput,
				}
			}
			if action, ok := firstItem.JSON["action"].(string); ok {
				return &models.NodeDataIO{
					Text: action,
				}
			}
		}
	}

	return nil
}

// extractOutputData extracts output data from node execution.
func (t *Transformer) extractOutputData(nodeExec *NodeExecution, nodeType string) *models.NodeDataIO {
	if len(nodeExec.Data.Main) == 0 {
		return nil
	}

	// Collect all output items
	var outputTexts []string
	var extraData map[string]interface{}

	for _, outputBranch := range nodeExec.Data.Main {
		for _, item := range outputBranch {
			// Check for text field first
			if item.Text != "" {
				outputTexts = append(outputTexts, item.Text)
				continue
			}

			// Check for output/response field in JSON
			if output, ok := item.JSON["output"].(string); ok {
				outputTexts = append(outputTexts, output)
				continue
			}

			// Check for text field in JSON
			if text, ok := item.JSON["text"].(string); ok {
				outputTexts = append(outputTexts, text)
				continue
			}

			// Check for response field in JSON
			if response, ok := item.JSON["response"].(string); ok {
				outputTexts = append(outputTexts, response)
				continue
			}

			// Store any JSON data as extra
			if len(item.JSON) > 0 {
				if extraData == nil {
					extraData = make(map[string]interface{})
				}
				// Flatten into extra data
				for k, v := range item.JSON {
					extraData[k] = v
				}
			}
		}
	}

	// Build output
	if len(outputTexts) > 0 || len(extraData) > 0 {
		output := &models.NodeDataIO{}
		if len(outputTexts) == 1 {
			output.Text = outputTexts[0]
		} else if len(outputTexts) > 1 {
			output.Text = strings.Join(outputTexts, "\n")
		}
		if len(extraData) > 0 {
			output.ExtraData = extraData
		}
		return output
	}

	return nil
}

// buildNodeMetadata constructs metadata from node execution.
func (t *Transformer) buildNodeMetadata(nodeExec *NodeExecution, nodeType string, runIndex int) map[string]interface{} {
	metadata := make(map[string]interface{})

	// Add N8N-specific metadata
	metadata["n8n_node_type"] = nodeType
	metadata["run_index"] = runIndex

	// Add token usage if available (for LLM nodes)
	if nodeExec.Metadata != nil && nodeExec.Metadata.TokenUsage != nil {
		metadata["token_usage"] = map[string]interface{}{
			"prompt_tokens":     nodeExec.Metadata.TokenUsage.PromptTokens,
			"completion_tokens": nodeExec.Metadata.TokenUsage.CompletionTokens,
			"total_tokens":      nodeExec.Metadata.TokenUsage.TotalTokens,
		}
	}

	// Add sub-execution info if available
	if nodeExec.Metadata != nil && nodeExec.Metadata.SubExecution != nil {
		metadata["sub_execution"] = map[string]interface{}{
			"workflow_id":  nodeExec.Metadata.SubExecution.WorkflowID,
			"execution_id": nodeExec.Metadata.SubExecution.ExecutionID,
		}
	}

	// Add error information if present
	if nodeExec.Error != nil {
		metadata["error"] = map[string]interface{}{
			"name":        nodeExec.Error.Name,
			"message":     nodeExec.Error.Message,
			"description": nodeExec.Error.Description,
		}
	}

	// Add source information
	if len(nodeExec.Source) > 0 {
		sources := make([]map[string]interface{}, len(nodeExec.Source))
		for i, src := range nodeExec.Source {
			sources[i] = map[string]interface{}{
				"previous_node":        src.PreviousNode,
				"previous_node_run":    src.PreviousNodeRun,
				"previous_node_output": src.PreviousNodeOutput,
			}
		}
		metadata["sources"] = sources
	}

	return metadata
}

// ExtractSessionID extracts the session ID from an execution response.
// Session ID is typically found in the chat trigger node data.
func (t *Transformer) ExtractSessionID(execution *ExecutionResponse) string {
	if execution == nil || execution.Data == nil || execution.Data.ResultData == nil {
		return ""
	}

	runData := execution.Data.ResultData.RunData
	if runData == nil {
		return ""
	}

	// Look for chat trigger node
	for nodeName, nodeExecutions := range runData {
		if strings.Contains(strings.ToLower(nodeName), "chat") || strings.Contains(strings.ToLower(nodeName), "trigger") {
			for _, nodeExec := range nodeExecutions {
				if len(nodeExec.Data.Main) > 0 && len(nodeExec.Data.Main[0]) > 0 {
					firstItem := nodeExec.Data.Main[0][0]
					if sessionID, ok := firstItem.JSON["sessionId"].(string); ok {
						return sessionID
					}
				}
			}
		}
	}

	return ""
}
