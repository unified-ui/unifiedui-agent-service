// Package foundry provides Microsoft Foundry trace import functionality.
package foundry

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/unifiedui/agent-service/internal/domain/models"
)

// Transformer transforms Foundry conversation items into TraceNodes.
type Transformer struct{}

// NewTransformer creates a new Foundry transformer.
func NewTransformer() *Transformer {
	return &Transformer{}
}

// Transform converts Foundry conversation items into a hierarchical TraceNode structure.
// The transformation follows these rules:
//   - Items are grouped by response_id to form "turns"
//   - SendActivity workflow_action items become container nodes for their response_id group
//   - Items with same response_id become children of the SendActivity
//   - Items without response_id (user messages, standalone items) are root nodes
//   - mcp_call, mcp_approval_request, mcp_approval_response are grouped by approval_request_id
//   - The chronological order is preserved (oldest to newest)
func (t *Transformer) Transform(items []ConversationItem, createdBy string) []models.TraceNode {
	if len(items) == 0 {
		return []models.TraceNode{}
	}

	// Reverse items to get chronological order (API returns newest first)
	reversedItems := make([]ConversationItem, len(items))
	for i, item := range items {
		reversedItems[len(items)-1-i] = item
	}

	// Group items by response_id for turn-based grouping
	responseGroups := t.groupByResponseID(reversedItems)

	// Build index maps for relationship resolution
	mcpApprovalGroups := t.groupByApprovalRequestID(reversedItems)

	// Find SendActivity containers for each response_id
	sendActivityContainers := t.findSendActivityContainers(reversedItems)

	// Transform into trace nodes with hierarchy
	nodes := t.buildTraceNodesWithHierarchy(reversedItems, responseGroups, mcpApprovalGroups, sendActivityContainers, createdBy)

	return nodes
}

// TransformInterface implements TraceTransformer interface.
func (t *Transformer) TransformInterface(items interface{}, createdBy string) []models.TraceNode {
	if convItems, ok := items.([]ConversationItem); ok {
		return t.Transform(convItems, createdBy)
	}
	return []models.TraceNode{}
}

// groupByResponseID groups items by their response_id.
func (t *Transformer) groupByResponseID(items []ConversationItem) map[string][]ConversationItem {
	groups := make(map[string][]ConversationItem)

	for _, item := range items {
		responseID := t.extractResponseID(item)
		if responseID != "" {
			groups[responseID] = append(groups[responseID], item)
		}
	}

	return groups
}

// groupByApprovalRequestID groups MCP items by their approval_request_id.
func (t *Transformer) groupByApprovalRequestID(items []ConversationItem) map[string][]ConversationItem {
	groups := make(map[string][]ConversationItem)

	for _, item := range items {
		if item.ApprovalRequestID != "" {
			groups[item.ApprovalRequestID] = append(groups[item.ApprovalRequestID], item)
		}
	}

	return groups
}

// extractResponseID extracts the response_id from an item's created_by field.
func (t *Transformer) extractResponseID(item ConversationItem) string {
	if item.CreatedBy == nil {
		return ""
	}

	if responseID, ok := item.CreatedBy["response_id"].(string); ok {
		return responseID
	}

	return ""
}

// findSendActivityContainers finds all SendActivity workflow_actions and maps their response_id to the item.
func (t *Transformer) findSendActivityContainers(items []ConversationItem) map[string]ConversationItem {
	containers := make(map[string]ConversationItem)

	for _, item := range items {
		if item.Type == "workflow_action" && item.Kind == "SendActivity" {
			responseID := t.extractResponseID(item)
			if responseID != "" {
				containers[responseID] = item
			}
		}
	}

	return containers
}

// buildTraceNodesWithHierarchy builds the hierarchical trace node structure.
func (t *Transformer) buildTraceNodesWithHierarchy(
	items []ConversationItem,
	responseGroups map[string][]ConversationItem,
	mcpApprovalGroups map[string][]ConversationItem,
	sendActivityContainers map[string]ConversationItem,
	createdBy string,
) []models.TraceNode {
	var nodes []models.TraceNode
	processedIDs := make(map[string]bool)

	for _, item := range items {
		if processedIDs[item.ID] {
			continue
		}

		responseID := t.extractResponseID(item)

		// Check if this item belongs to a SendActivity container
		if responseID != "" {
			if containerItem, hasContainer := sendActivityContainers[responseID]; hasContainer {
				if item.ID != containerItem.ID {
					continue
				}
			}
		}

		switch item.Type {
		case "message":
			node := t.transformMessage(item, createdBy)
			nodes = append(nodes, node)
			processedIDs[item.ID] = true

		case "workflow_action":
			if item.Kind == "SendActivity" && responseID != "" {
				node := t.transformSendActivityWithChildren(item, responseGroups, mcpApprovalGroups, processedIDs, createdBy)
				nodes = append(nodes, node)
			} else {
				node := t.transformWorkflowAction(item, createdBy)
				nodes = append(nodes, node)
				processedIDs[item.ID] = true
			}

		case "mcp_approval_request":
			node := t.transformMCPGroup(item, mcpApprovalGroups, createdBy)
			nodes = append(nodes, node)
			processedIDs[item.ID] = true
			if relatedItems, ok := mcpApprovalGroups[item.ID]; ok {
				for _, related := range relatedItems {
					processedIDs[related.ID] = true
				}
			}

		case "mcp_call":
			if item.ApprovalRequestID == "" || !t.hasApprovalRequest(items, item.ApprovalRequestID) {
				node := t.transformMCPCall(item, createdBy)
				nodes = append(nodes, node)
			}
			processedIDs[item.ID] = true

		case "mcp_approval_response":
			processedIDs[item.ID] = true

		case "mcp_list_tools":
			node := t.transformMCPListTools(item, createdBy)
			nodes = append(nodes, node)
			processedIDs[item.ID] = true

		default:
			node := t.transformUnknown(item, createdBy)
			nodes = append(nodes, node)
			processedIDs[item.ID] = true
		}
	}

	return nodes
}

// transformSendActivityWithChildren transforms a SendActivity into a container node.
func (t *Transformer) transformSendActivityWithChildren(
	sendActivity ConversationItem,
	responseGroups map[string][]ConversationItem,
	mcpApprovalGroups map[string][]ConversationItem,
	processedIDs map[string]bool,
	createdBy string,
) models.TraceNode {
	now := time.Now().UTC()
	responseID := t.extractResponseID(sendActivity)

	processedIDs[sendActivity.ID] = true

	var childNodes []models.TraceNode
	if groupItems, ok := responseGroups[responseID]; ok {
		for _, groupItem := range groupItems {
			if groupItem.ID == sendActivity.ID {
				continue
			}
			if processedIDs[groupItem.ID] {
				continue
			}

			var childNode models.TraceNode
			switch groupItem.Type {
			case "message":
				childNode = t.transformMessage(groupItem, createdBy)
			case "workflow_action":
				childNode = t.transformWorkflowAction(groupItem, createdBy)
			case "mcp_approval_request":
				childNode = t.transformMCPGroup(groupItem, mcpApprovalGroups, createdBy)
				if relatedItems, ok := mcpApprovalGroups[groupItem.ID]; ok {
					for _, related := range relatedItems {
						processedIDs[related.ID] = true
					}
				}
			case "mcp_call":
				childNode = t.transformMCPCall(groupItem, createdBy)
			case "mcp_list_tools":
				childNode = t.transformMCPListTools(groupItem, createdBy)
			default:
				childNode = t.transformUnknown(groupItem, createdBy)
			}

			childNodes = append(childNodes, childNode)
			processedIDs[groupItem.ID] = true
		}
	}

	metadata := map[string]interface{}{
		"kind": sendActivity.Kind,
	}
	if sendActivity.ActionID != "" {
		metadata["action_id"] = sendActivity.ActionID
	}
	if sendActivity.PreviousActionID != "" {
		metadata["previous_action_id"] = sendActivity.PreviousActionID
	}
	if sendActivity.CreatedBy != nil {
		metadata["created_by"] = sendActivity.CreatedBy
	}

	node := models.TraceNode{
		ID:          "node_" + uuid.New().String(),
		Name:        "SendActivity",
		Type:        models.NodeTypeWorkflow,
		ReferenceID: sendActivity.ID,
		Status:      t.mapStatus(sendActivity.Status),
		Data: &models.NodeData{
			Output: &models.NodeDataIO{
				Metadata: metadata,
			},
		},
		Metadata:  t.buildWorkflowMetadata(sendActivity),
		Nodes:     childNodes,
		Logs:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: createdBy,
		UpdatedBy: createdBy,
	}

	return node
}

// hasApprovalRequest checks if there's an approval request item with the given ID.
func (t *Transformer) hasApprovalRequest(items []ConversationItem, approvalRequestID string) bool {
	for _, item := range items {
		if item.Type == "mcp_approval_request" && item.ID == approvalRequestID {
			return true
		}
	}
	return false
}

// transformMessage transforms a message item into a TraceNode.
func (t *Transformer) transformMessage(item ConversationItem, createdBy string) models.TraceNode {
	now := time.Now().UTC()

	inputText, outputText := t.extractMessageContent(item)

	name := "Message"
	if item.Role == "user" {
		name = "User Message"
	} else if item.Role == "assistant" {
		name = "Assistant Response"
	}

	node := models.TraceNode{
		ID:          "node_" + uuid.New().String(),
		Name:        name,
		Type:        models.NodeTypeLLM,
		ReferenceID: item.ID,
		Status:      t.mapStatus(item.Status),
		Data: &models.NodeData{
			Input: &models.NodeDataIO{
				Text: inputText,
				Metadata: map[string]interface{}{
					"role": item.Role,
					"type": item.Type,
				},
			},
			Output: &models.NodeDataIO{
				Text: outputText,
				Metadata: map[string]interface{}{
					"role": item.Role,
					"type": item.Type,
				},
			},
		},
		Metadata:  t.buildMessageMetadata(item),
		Nodes:     []models.TraceNode{},
		Logs:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: createdBy,
		UpdatedBy: createdBy,
	}

	return node
}

// transformWorkflowAction transforms a workflow_action item into a TraceNode.
func (t *Transformer) transformWorkflowAction(item ConversationItem, createdBy string) models.TraceNode {
	now := time.Now().UTC()

	name := "Workflow Action"
	if item.Kind != "" {
		name = t.formatKindAsName(item.Kind)
	}

	node := models.TraceNode{
		ID:          "node_" + uuid.New().String(),
		Name:        name,
		Type:        models.NodeTypeWorkflow,
		ReferenceID: item.ID,
		Status:      t.mapStatus(item.Status),
		Data: &models.NodeData{
			Input: &models.NodeDataIO{
				Metadata: map[string]interface{}{
					"kind":               item.Kind,
					"action_id":          item.ActionID,
					"parent_action_id":   item.ParentActionID,
					"previous_action_id": item.PreviousActionID,
				},
			},
		},
		Metadata:  t.buildWorkflowMetadata(item),
		Nodes:     []models.TraceNode{},
		Logs:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: createdBy,
		UpdatedBy: createdBy,
	}

	return node
}

// transformMCPGroup transforms an MCP approval request and its related items into a TraceNode.
func (t *Transformer) transformMCPGroup(
	approvalRequest ConversationItem,
	mcpApprovalGroups map[string][]ConversationItem,
	createdBy string,
) models.TraceNode {
	now := time.Now().UTC()

	relatedItems := mcpApprovalGroups[approvalRequest.ID]

	var mcpCall *ConversationItem
	var mcpResponse *ConversationItem

	for i := range relatedItems {
		switch relatedItems[i].Type {
		case "mcp_call":
			mcpCall = &relatedItems[i]
		case "mcp_approval_response":
			mcpResponse = &relatedItems[i]
		}
	}

	name := "MCP Tool Call"
	if approvalRequest.Name != "" {
		name = approvalRequest.Name
	}

	inputText := approvalRequest.Arguments
	outputText := ""
	if mcpCall != nil && mcpCall.Output != "" {
		outputText = mcpCall.Output
	}

	status := models.NodeStatusCompleted
	if mcpResponse != nil && mcpResponse.Approve != nil && !*mcpResponse.Approve {
		status = models.NodeStatusCancelled
	}

	var subNodes []models.TraceNode
	subNodes = append(subNodes, t.transformMCPApprovalRequest(approvalRequest, createdBy))
	if mcpResponse != nil {
		subNodes = append(subNodes, t.transformMCPApprovalResponse(*mcpResponse, createdBy))
	}
	if mcpCall != nil {
		subNodes = append(subNodes, t.transformMCPCall(*mcpCall, createdBy))
	}

	node := models.TraceNode{
		ID:          "node_" + uuid.New().String(),
		Name:        name,
		Type:        models.NodeTypeTool,
		ReferenceID: approvalRequest.ID,
		Status:      status,
		Data: &models.NodeData{
			Input: &models.NodeDataIO{
				Text: inputText,
				Metadata: map[string]interface{}{
					"server_label": approvalRequest.ServerLabel,
					"tool_name":    approvalRequest.Name,
				},
			},
			Output: &models.NodeDataIO{
				Text: outputText,
			},
		},
		Metadata:  t.buildMCPMetadata(approvalRequest),
		Nodes:     subNodes,
		Logs:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: createdBy,
		UpdatedBy: createdBy,
	}

	return node
}

// transformMCPApprovalRequest transforms an mcp_approval_request into a TraceNode.
func (t *Transformer) transformMCPApprovalRequest(item ConversationItem, createdBy string) models.TraceNode {
	now := time.Now().UTC()

	return models.TraceNode{
		ID:          "node_" + uuid.New().String(),
		Name:        "Approval Request: " + item.Name,
		Type:        models.NodeTypeTool,
		ReferenceID: item.ID,
		Status:      models.NodeStatusCompleted,
		Data: &models.NodeData{
			Input: &models.NodeDataIO{
				Text: item.Arguments,
				Metadata: map[string]interface{}{
					"server_label": item.ServerLabel,
					"tool_name":    item.Name,
				},
			},
		},
		Metadata:  t.buildMCPMetadata(item),
		Nodes:     []models.TraceNode{},
		Logs:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: createdBy,
		UpdatedBy: createdBy,
	}
}

// transformMCPApprovalResponse transforms an mcp_approval_response into a TraceNode.
func (t *Transformer) transformMCPApprovalResponse(item ConversationItem, createdBy string) models.TraceNode {
	now := time.Now().UTC()

	status := models.NodeStatusCompleted
	approved := false
	if item.Approve != nil {
		approved = *item.Approve
		if !approved {
			status = models.NodeStatusCancelled
		}
	}

	name := "Approval Response: Denied"
	if approved {
		name = "Approval Response: Approved"
	}

	return models.TraceNode{
		ID:          "node_" + uuid.New().String(),
		Name:        name,
		Type:        models.NodeTypeTool,
		ReferenceID: item.ID,
		Status:      status,
		Data: &models.NodeData{
			Output: &models.NodeDataIO{
				Metadata: map[string]interface{}{
					"approved":            approved,
					"approval_request_id": item.ApprovalRequestID,
				},
			},
		},
		Metadata: map[string]interface{}{
			"partition_key":       item.PartitionKey,
			"approval_request_id": item.ApprovalRequestID,
		},
		Nodes:     []models.TraceNode{},
		Logs:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: createdBy,
		UpdatedBy: createdBy,
	}
}

// transformMCPCall transforms an mcp_call into a TraceNode.
func (t *Transformer) transformMCPCall(item ConversationItem, createdBy string) models.TraceNode {
	now := time.Now().UTC()

	name := "MCP Call"
	if item.Name != "" {
		name = "MCP Call: " + item.Name
	}

	return models.TraceNode{
		ID:          "node_" + uuid.New().String(),
		Name:        name,
		Type:        models.NodeTypeTool,
		ReferenceID: item.ID,
		Status:      t.mapStatus(item.Status),
		Data: &models.NodeData{
			Input: &models.NodeDataIO{
				Text: item.Arguments,
				Metadata: map[string]interface{}{
					"server_label": item.ServerLabel,
					"tool_name":    item.Name,
				},
			},
			Output: &models.NodeDataIO{
				Text: item.Output,
			},
		},
		Metadata:  t.buildMCPMetadata(item),
		Nodes:     []models.TraceNode{},
		Logs:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: createdBy,
		UpdatedBy: createdBy,
	}
}

// transformMCPListTools transforms an mcp_list_tools into a TraceNode.
func (t *Transformer) transformMCPListTools(item ConversationItem, createdBy string) models.TraceNode {
	now := time.Now().UTC()

	toolsJSON := ""
	if item.Content != nil {
		if data, err := json.Marshal(item.Content); err == nil {
			toolsJSON = string(data)
		}
	}

	return models.TraceNode{
		ID:          "node_" + uuid.New().String(),
		Name:        "MCP List Tools: " + item.ServerLabel,
		Type:        models.NodeTypeTool,
		ReferenceID: item.ID,
		Status:      models.NodeStatusCompleted,
		Data: &models.NodeData{
			Input: &models.NodeDataIO{
				Metadata: map[string]interface{}{
					"server_label": item.ServerLabel,
				},
			},
			Output: &models.NodeDataIO{
				Text: toolsJSON,
			},
		},
		Metadata: map[string]interface{}{
			"partition_key": item.PartitionKey,
			"response_id":   t.extractResponseID(item),
		},
		Nodes:     []models.TraceNode{},
		Logs:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: createdBy,
		UpdatedBy: createdBy,
	}
}

// transformUnknown transforms an unknown item type into a TraceNode.
func (t *Transformer) transformUnknown(item ConversationItem, createdBy string) models.TraceNode {
	now := time.Now().UTC()

	itemJSON := ""
	if data, err := json.Marshal(item); err == nil {
		itemJSON = string(data)
	}

	return models.TraceNode{
		ID:          "node_" + uuid.New().String(),
		Name:        "Unknown: " + item.Type,
		Type:        models.NodeTypeCustom,
		ReferenceID: item.ID,
		Status:      t.mapStatus(item.Status),
		Data: &models.NodeData{
			Input: &models.NodeDataIO{
				Text: itemJSON,
				Metadata: map[string]interface{}{
					"original_type": item.Type,
				},
			},
		},
		Metadata: map[string]interface{}{
			"partition_key": item.PartitionKey,
			"original_type": item.Type,
		},
		Nodes:     []models.TraceNode{},
		Logs:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: createdBy,
		UpdatedBy: createdBy,
	}
}

// extractMessageContent extracts input and output text from message content.
func (t *Transformer) extractMessageContent(item ConversationItem) (inputText, outputText string) {
	if item.Content == nil {
		return "", ""
	}

	var texts []string
	for _, c := range item.Content {
		contentMap, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		text, ok := contentMap["text"].(string)
		if ok && text != "" {
			texts = append(texts, text)
		}
	}

	combinedText := strings.Join(texts, "\n")

	if item.Role == "user" {
		return combinedText, ""
	}
	return "", combinedText
}

// mapStatus maps Foundry status to NodeStatus.
func (t *Transformer) mapStatus(status string) models.NodeStatus {
	switch status {
	case "completed":
		return models.NodeStatusCompleted
	case "failed":
		return models.NodeStatusFailed
	case "cancelled":
		return models.NodeStatusCancelled
	case "pending":
		return models.NodeStatusPending
	case "running", "in_progress":
		return models.NodeStatusRunning
	default:
		if status == "" {
			return models.NodeStatusCompleted
		}
		return models.NodeStatusCompleted
	}
}

// formatKindAsName converts a workflow kind to a readable name.
func (t *Transformer) formatKindAsName(kind string) string {
	var result strings.Builder
	for i, r := range kind {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune(' ')
		}
		result.WriteRune(r)
	}
	return result.String()
}

// buildMessageMetadata builds metadata for a message node.
func (t *Transformer) buildMessageMetadata(item ConversationItem) map[string]interface{} {
	metadata := map[string]interface{}{
		"partition_key": item.PartitionKey,
	}

	if responseID := t.extractResponseID(item); responseID != "" {
		metadata["response_id"] = responseID
	}

	if item.CreatedBy != nil {
		if agent, ok := item.CreatedBy["agent"].(map[string]interface{}); ok {
			metadata["agent"] = agent
		}
	}

	return metadata
}

// buildWorkflowMetadata builds metadata for a workflow action node.
func (t *Transformer) buildWorkflowMetadata(item ConversationItem) map[string]interface{} {
	metadata := map[string]interface{}{
		"action_id":          item.ActionID,
		"parent_action_id":   item.ParentActionID,
		"previous_action_id": item.PreviousActionID,
		"kind":               item.Kind,
	}

	if responseID := t.extractResponseID(item); responseID != "" {
		metadata["response_id"] = responseID
	}

	if item.CreatedBy != nil {
		if agent, ok := item.CreatedBy["agent"].(map[string]interface{}); ok {
			metadata["agent"] = agent
		}
	}

	return metadata
}

// buildMCPMetadata builds metadata for an MCP node.
func (t *Transformer) buildMCPMetadata(item ConversationItem) map[string]interface{} {
	metadata := map[string]interface{}{
		"partition_key":       item.PartitionKey,
		"server_label":        item.ServerLabel,
		"approval_request_id": item.ApprovalRequestID,
	}

	if responseID := t.extractResponseID(item); responseID != "" {
		metadata["response_id"] = responseID
	}

	if item.CreatedBy != nil {
		if agent, ok := item.CreatedBy["agent"].(map[string]interface{}); ok {
			metadata["agent"] = agent
		}
	}

	return metadata
}

// SortNodesByTime sorts nodes by their creation time.
func SortNodesByTime(nodes []models.TraceNode) {
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].CreatedAt.Before(nodes[j].CreatedAt)
	})
}
