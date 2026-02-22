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
// Sub-nodes connected via non-main connections (ai_languageModel, ai_tool, ai_memory)
// are nested as children of their parent node. Top-level nodes are sorted chronologically.
func (t *Transformer) TransformExecution(execution *ExecutionResponse, createdBy string) []models.TraceNode {
	if execution == nil || execution.Data == nil || execution.Data.ResultData == nil {
		return []models.TraceNode{}
	}

	runData := execution.Data.ResultData.RunData
	if len(runData) == 0 {
		return []models.TraceNode{}
	}

	workflowNodeMap := t.buildWorkflowNodeMap(execution.WorkflowData)
	parentOf, connectionTypes := t.parseConnectionGraph(execution.WorkflowData)

	nodesByName := make(map[string][]models.TraceNode)
	for nodeName, nodeExecutions := range runData {
		nodeType := ""
		if wfNode, exists := workflowNodeMap[nodeName]; exists {
			nodeType = wfNode.Type
		}

		for runIndex := range nodeExecutions {
			traceNode := t.transformNodeExecution(nodeName, nodeType, runIndex, &nodeExecutions[runIndex], createdBy)
			if connType, isSubNode := connectionTypes[nodeName]; isSubNode {
				traceNode.Metadata["connection_type"] = connType
			}
			nodesByName[nodeName] = append(nodesByName[nodeName], traceNode)
		}
	}

	subNodeNames := make(map[string]bool)
	for childName, parentName := range parentOf {
		children := nodesByName[childName]
		if len(children) == 0 {
			continue
		}
		parentNodes := nodesByName[parentName]
		if len(parentNodes) == 0 {
			continue
		}
		parentNodes[0].Nodes = append(parentNodes[0].Nodes, children...)
		subNodeNames[childName] = true
	}

	t.sortChildNodes(nodesByName, subNodeNames)

	var topLevelNodes []models.TraceNode
	for nodeName, nodes := range nodesByName {
		if subNodeNames[nodeName] {
			continue
		}
		topLevelNodes = append(topLevelNodes, nodes...)
	}

	sort.Slice(topLevelNodes, func(i, j int) bool {
		if topLevelNodes[i].StartAt == nil {
			return true
		}
		if topLevelNodes[j].StartAt == nil {
			return false
		}
		return topLevelNodes[i].StartAt.Before(*topLevelNodes[j].StartAt)
	})

	return topLevelNodes
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

	for i := range workflowData.Nodes {
		nodeMap[workflowData.Nodes[i].Name] = workflowData.Nodes[i]
	}

	return nodeMap
}

// parseConnectionGraph extracts parent-child relationships from WorkflowData.Connections.
// Non-main connections (ai_languageModel, ai_tool, ai_memory) indicate sub-node
// relationships where the source node is a child of the target node.
func (t *Transformer) parseConnectionGraph(workflowData *WorkflowData) (parentOf, connectionTypes map[string]string) {
	parentOf = make(map[string]string)
	connectionTypes = make(map[string]string)

	if workflowData == nil || workflowData.Connections == nil {
		return
	}

	for sourceName, destTypesRaw := range workflowData.Connections {
		destTypes, ok := destTypesRaw.(map[string]interface{})
		if !ok {
			continue
		}

		for connType, branchesRaw := range destTypes {
			if connType == "main" {
				continue
			}

			branches, ok := branchesRaw.([]interface{})
			if !ok {
				continue
			}

			for _, branchRaw := range branches {
				branch, ok := branchRaw.([]interface{})
				if !ok {
					continue
				}

				for _, connRaw := range branch {
					conn, ok := connRaw.(map[string]interface{})
					if !ok {
						continue
					}

					targetNode, _ := conn["node"].(string)
					if targetNode == "" {
						continue
					}

					parentOf[sourceName] = targetNode
					connectionTypes[sourceName] = connType
				}
			}
		}
	}

	return
}

// sortChildNodes sorts the child nodes of each parent node by start time.
func (t *Transformer) sortChildNodes(nodesByName map[string][]models.TraceNode, subNodeNames map[string]bool) {
	for nodeName, nodes := range nodesByName {
		if subNodeNames[nodeName] {
			continue
		}
		for i := range nodes {
			if len(nodes[i].Nodes) <= 1 {
				continue
			}
			sort.Slice(nodes[i].Nodes, func(a, b int) bool {
				if nodes[i].Nodes[a].StartAt == nil {
					return true
				}
				if nodes[i].Nodes[b].StartAt == nil {
					return false
				}
				return nodes[i].Nodes[a].StartAt.Before(*nodes[i].Nodes[b].StartAt)
			})
		}
	}
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
	suffix := extractNodeSuffix(n8nType)

	switch {
	case strings.HasPrefix(suffix, "lmChat"):
		return models.NodeTypeLLM
	case strings.HasPrefix(suffix, "embeddings"):
		return models.NodeTypeEmbedding
	case strings.HasPrefix(suffix, "memory"):
		return models.NodeTypeMemory
	case strings.HasPrefix(suffix, "vectorStore"):
		return models.NodeTypeVectorStore
	case strings.HasPrefix(suffix, "outputParser"):
		return models.NodeTypeOutputParser
	case strings.HasPrefix(suffix, "document"):
		return models.NodeTypeDocument
	case strings.HasPrefix(suffix, "textSplitter"):
		return models.NodeTypeTextSplitter
	case strings.HasPrefix(suffix, "retriever"):
		return models.NodeTypeRetriever
	case strings.HasSuffix(suffix, "Trigger") || suffix == "webhook":
		return models.NodeTypeWorkflow
	case suffix == "agent" || suffix == "information-extractor" ||
		suffix == "text-classifier" || suffix == "textClassifier" ||
		suffix == "sentimentAnalysis":
		return models.NodeTypeAgent
	case strings.HasPrefix(suffix, "chain"):
		return models.NodeTypeChain
	case strings.HasPrefix(suffix, "tool"):
		return models.NodeTypeTool
	case suffix == "httpRequest":
		return models.NodeTypeHTTP
	case suffix == "code":
		return models.NodeTypeCode
	case strings.HasPrefix(suffix, "function"):
		return models.NodeTypeCode
	case suffix == "switch" || suffix == "if" || suffix == "filter":
		return models.NodeTypeConditional
	case suffix == "splitInBatches":
		return models.NodeTypeLoop
	case suffix == "postgres" || suffix == "mongoDb" || suffix == "mySql" ||
		suffix == "redis" || suffix == "executeCommand" || suffix == "readWriteFile":
		return models.NodeTypeTool
	case suffix == "merge" || suffix == "executeWorkflow" || suffix == "respondToWebhook":
		return models.NodeTypeWorkflow
	default:
		return models.NodeTypeCustom
	}
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
func (t *Transformer) extractOutputData(nodeExec *NodeExecution, _ string) *models.NodeDataIO {
	outputItems := nodeExec.Data.GetOutputItems()
	if len(outputItems) == 0 {
		return nil
	}

	// Collect all output items
	var outputTexts []string
	var extraData map[string]interface{}

	for _, outputBranch := range outputItems {
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

			// Check for response field in JSON (string)
			if response, ok := item.JSON["response"].(string); ok {
				outputTexts = append(outputTexts, response)
				continue
			}

			// Check for LLM response structure: response.generations[0][0].text
			if responseMap, ok := item.JSON["response"].(map[string]interface{}); ok {
				if text := extractLLMResponseText(responseMap); text != "" {
					outputTexts = append(outputTexts, text)
					continue
				}
			}

			// Check for structured output (map)
			if output, ok := item.JSON["output"].(map[string]interface{}); ok {
				if extraData == nil {
					extraData = make(map[string]interface{})
				}
				extraData["output"] = output
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

// extractLLMResponseText extracts text from the N8N LLM response.generations structure.
func extractLLMResponseText(response map[string]interface{}) string {
	generations, ok := response["generations"].([]interface{})
	if !ok || len(generations) == 0 {
		return ""
	}
	firstGen, ok := generations[0].([]interface{})
	if !ok || len(firstGen) == 0 {
		return ""
	}
	genItem, ok := firstGen[0].(map[string]interface{})
	if !ok {
		return ""
	}
	text, _ := genItem["text"].(string)
	return text
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
			for j := range nodeExecutions {
				if len(nodeExecutions[j].Data.Main) > 0 && len(nodeExecutions[j].Data.Main[0]) > 0 {
					firstItem := nodeExecutions[j].Data.Main[0][0]
					if sessionID, ok := firstItem.JSON["sessionId"].(string); ok {
						return sessionID
					}
				}
			}
		}
	}

	return ""
}
