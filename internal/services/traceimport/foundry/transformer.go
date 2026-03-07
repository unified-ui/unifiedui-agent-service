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

// isFunctionCallType returns true if the item type is a function call (local or remote).
func isFunctionCallType(itemType string) bool {
	return itemType == "function_call" || itemType == "remote_function_call"
}

// isFunctionCallOutputType returns true if the item type is a function call output (local or remote).
func isFunctionCallOutputType(itemType string) bool {
	return itemType == "function_call_output" || itemType == "remote_function_call_output"
}

// Transformer transforms Foundry conversation items into TraceNodes.
type Transformer struct{}

// NewTransformer creates a new Foundry transformer.
func NewTransformer() *Transformer {
	return &Transformer{}
}

// Transform converts Foundry conversation items into a hierarchical TraceNode structure.
//
// The transformation uses the action_id/previous_action_id DAG for sequencing and assigns
// messages to their producing workflow actions:
//   - Workflow actions become top-level nodes in chronological order
//   - Messages with response_id matching a SendActivity/Question become children of that action
//   - Sub-agent messages (no response_id) are assigned to the nearest preceding InvokeAzureAgent
//   - User messages are always top-level
//   - MCP groups (approval_request + response + call) are children of their triggering action
//   - EndConversation and other actions without children are standalone top-level nodes
func (t *Transformer) Transform(items []ConversationItem, createdBy string) []models.TraceNode {
	if len(items) == 0 {
		return []models.TraceNode{}
	}

	chronological := make([]ConversationItem, len(items))
	for i := range items {
		chronological[len(items)-1-i] = items[i]
	}

	mcpApprovalGroups := t.groupByApprovalRequestID(chronological)
	functionCallOutputs := t.groupFunctionCallOutputsByCallID(chronological)
	functionCallsByRespID := t.groupFunctionCallsByResponseID(chronological)
	mcpItemsByRespID := t.groupMCPItemsByResponseID(chronological)
	assignment := t.buildActionAssignment(chronological)

	return t.buildNodeList(chronological, mcpApprovalGroups, functionCallOutputs, functionCallsByRespID, mcpItemsByRespID, assignment, createdBy)
}

// TransformInterface implements TraceTransformer interface.
func (t *Transformer) TransformInterface(items interface{}, createdBy string) []models.TraceNode {
	if convItems, ok := items.([]ConversationItem); ok {
		return t.Transform(convItems, createdBy)
	}
	return []models.TraceNode{}
}

// actionAssignment tracks which items become children of which workflow actions.
type actionAssignment struct {
	messageParent      map[string]int
	mcpParent          map[string]int
	mcpCallParent      map[string]int
	functionCallParent map[string]int
}

// buildActionAssignment determines which messages and MCP items become children of which workflow actions.
func (t *Transformer) buildActionAssignment(items []ConversationItem) actionAssignment {
	a := actionAssignment{
		messageParent:      make(map[string]int),
		mcpParent:          make(map[string]int),
		mcpCallParent:      make(map[string]int),
		functionCallParent: make(map[string]int),
	}

	messageActionByRespID := make(map[string]int)
	for i := range items {
		if items[i].Type == "workflow_action" && t.isMessageProducingAction(items[i]) {
			if respID := t.extractResponseID(items[i]); respID != "" {
				messageActionByRespID[respID] = i
			}
		}
	}

	actionByRespID := make(map[string]int)
	for i := range items {
		if items[i].Type == "workflow_action" {
			if respID := t.extractResponseID(items[i]); respID != "" {
				if _, exists := actionByRespID[respID]; !exists {
					actionByRespID[respID] = i
				}
			}
		}
	}

	lastInvokeIdx := -1
	for i := range items {
		if items[i].Type == "workflow_action" && t.isInvokeAction(items[i]) {
			lastInvokeIdx = i
		}

		if items[i].Type != "message" || items[i].Role == "user" {
			continue
		}

		respID := t.extractResponseID(items[i])
		if respID != "" {
			if actionIdx, ok := messageActionByRespID[respID]; ok {
				a.messageParent[items[i].ID] = actionIdx
				continue
			}
		}

		if respID == "" && lastInvokeIdx >= 0 && t.isSubAgentMessage(items[i]) {
			a.messageParent[items[i].ID] = lastInvokeIdx
		}
	}

	for i := range items {
		if items[i].Type == "mcp_approval_request" {
			if respID := t.extractResponseID(items[i]); respID != "" {
				if actionIdx, ok := actionByRespID[respID]; ok {
					a.mcpParent[items[i].ID] = actionIdx
				}
			}
		}
	}

	for i := range items {
		if items[i].Type == "mcp_call" && items[i].ApprovalRequestID == "" {
			if respID := t.extractResponseID(items[i]); respID != "" {
				if actionIdx, ok := actionByRespID[respID]; ok {
					a.mcpCallParent[items[i].ID] = actionIdx
				}
			}
		}
	}

	// Assign function_call / remote_function_call items to workflow actions
	for i := range items {
		if isFunctionCallType(items[i].Type) {
			if respID := t.extractResponseID(items[i]); respID != "" {
				if actionIdx, ok := actionByRespID[respID]; ok {
					a.functionCallParent[items[i].ID] = actionIdx
				}
			}
		}
	}

	return a
}

// buildNodeList creates the final list of top-level TraceNodes with children attached.
func (t *Transformer) buildNodeList(
	items []ConversationItem,
	mcpApprovalGroups map[string][]ConversationItem,
	functionCallOutputs map[string]*ConversationItem,
	functionCallsByRespID map[string][]ConversationItem,
	mcpItemsByRespID map[string][]ConversationItem,
	assignment actionAssignment,
	createdBy string,
) []models.TraceNode {
	actionChildren := make(map[int][]models.TraceNode)
	processedIDs := make(map[string]bool)

	for i := range items {
		if actionIdx, ok := assignment.messageParent[items[i].ID]; ok {
			actionChildren[actionIdx] = append(actionChildren[actionIdx], t.transformMessage(items[i], createdBy))
			processedIDs[items[i].ID] = true
		}
	}

	for i := range items {
		if items[i].Type != "mcp_approval_request" {
			continue
		}
		if actionIdx, ok := assignment.mcpParent[items[i].ID]; ok {
			actionChildren[actionIdx] = append(actionChildren[actionIdx], t.transformMCPGroup(items[i], mcpApprovalGroups, createdBy))
			processedIDs[items[i].ID] = true
			t.markMCPGroupProcessed(items[i].ID, mcpApprovalGroups, processedIDs)
		}
	}

	for i := range items {
		if items[i].Type != "mcp_call" || items[i].ApprovalRequestID != "" {
			continue
		}
		if actionIdx, ok := assignment.mcpCallParent[items[i].ID]; ok {
			actionChildren[actionIdx] = append(actionChildren[actionIdx], t.transformMCPCall(items[i], createdBy))
			processedIDs[items[i].ID] = true
		}
	}

	// Assign function_call / remote_function_call items to their parent workflow actions
	for i := range items {
		if !isFunctionCallType(items[i].Type) {
			continue
		}
		if actionIdx, ok := assignment.functionCallParent[items[i].ID]; ok {
			actionChildren[actionIdx] = append(actionChildren[actionIdx], t.transformFunctionCall(items[i], functionCallOutputs, createdBy))
			processedIDs[items[i].ID] = true
			t.markFunctionCallOutputProcessed(items[i], functionCallOutputs, processedIDs)
		}
	}

	var nodes []models.TraceNode

	// Build set of response_ids that have assistant messages.
	// Function calls and MCP items with these response_ids will be nested under the
	// assistant message instead of appearing as standalone top-level nodes.
	messageRespIDs := make(map[string]bool)
	for i := range items {
		if items[i].Type == "message" && items[i].Role != "user" {
			if respID := t.extractResponseID(items[i]); respID != "" {
				messageRespIDs[respID] = true
			}
		}
	}

	for i := range items {
		if processedIDs[items[i].ID] {
			continue
		}

		switch {
		case items[i].Type == "message":
			node := t.transformMessage(items[i], createdBy)
			// For assistant messages, attach function calls and MCP items with the same response_id as children
			if items[i].Role != "user" {
				if respID := t.extractResponseID(items[i]); respID != "" {
					// Attach function calls
					if fcItems, ok := functionCallsByRespID[respID]; ok {
						for _, fcItem := range fcItems {
							if !processedIDs[fcItem.ID] {
								node.Nodes = append(node.Nodes, t.transformFunctionCall(fcItem, functionCallOutputs, createdBy))
								processedIDs[fcItem.ID] = true
								t.markFunctionCallOutputProcessed(fcItem, functionCallOutputs, processedIDs)
							}
						}
					}
					// Attach MCP items (mcp_list_tools, mcp_call, mcp_approval_request)
					if mcpItems, ok := mcpItemsByRespID[respID]; ok {
						for _, mcpItem := range mcpItems {
							if !processedIDs[mcpItem.ID] {
								mcpNode := t.transformMCPItemForNesting(mcpItem, mcpApprovalGroups, createdBy)
								node.Nodes = append(node.Nodes, mcpNode)
								processedIDs[mcpItem.ID] = true
								if mcpItem.Type == "mcp_approval_request" {
									t.markMCPGroupProcessed(mcpItem.ID, mcpApprovalGroups, processedIDs)
								}
							}
						}
					}
				}
			}
			nodes = append(nodes, node)

		case items[i].Type == "workflow_action":
			node := t.transformWorkflowAction(items[i], createdBy)
			if children, ok := actionChildren[i]; ok {
				node.Nodes = children
			}
			nodes = append(nodes, node)

		case items[i].Type == "mcp_approval_request":
			// Skip if will be nested under an assistant message
			if respID := t.extractResponseID(items[i]); respID != "" && messageRespIDs[respID] {
				t.markMCPGroupProcessed(items[i].ID, mcpApprovalGroups, processedIDs)
				continue
			}
			nodes = append(nodes, t.transformMCPGroup(items[i], mcpApprovalGroups, createdBy))
			t.markMCPGroupProcessed(items[i].ID, mcpApprovalGroups, processedIDs)

		case items[i].Type == "mcp_call":
			// Skip if will be nested under an assistant message
			if respID := t.extractResponseID(items[i]); respID != "" && messageRespIDs[respID] {
				continue
			}
			if items[i].ApprovalRequestID == "" || !t.hasApprovalRequest(items, items[i].ApprovalRequestID) {
				nodes = append(nodes, t.transformMCPCall(items[i], createdBy))
			}

		case items[i].Type == "mcp_approval_response":
			// Handled as part of MCP group

		case items[i].Type == "mcp_list_tools":
			// Skip if will be nested under an assistant message
			if respID := t.extractResponseID(items[i]); respID != "" && messageRespIDs[respID] {
				continue
			}
			nodes = append(nodes, t.transformMCPListTools(items[i], createdBy))

		case isFunctionCallType(items[i].Type):
			// Skip if this function call will be nested under an assistant message
			if respID := t.extractResponseID(items[i]); respID != "" && messageRespIDs[respID] {
				// Mark the output as processed so it doesn't appear standalone either
				t.markFunctionCallOutputProcessed(items[i], functionCallOutputs, processedIDs)
				continue
			}
			// Standalone function_call not assigned to any workflow action or assistant message
			node := t.transformFunctionCall(items[i], functionCallOutputs, createdBy)
			nodes = append(nodes, node)
			t.markFunctionCallOutputProcessed(items[i], functionCallOutputs, processedIDs)

		case isFunctionCallOutputType(items[i].Type):
			// Standalone function_call_output (no matching function_call found)
			nodes = append(nodes, t.transformFunctionCallOutput(items[i], createdBy))

		default:
			nodes = append(nodes, t.transformUnknown(items[i], createdBy))
		}

		processedIDs[items[i].ID] = true
	}

	return nodes
}

// isMessageProducingAction returns true for workflow action kinds that produce visible messages.
func (t *Transformer) isMessageProducingAction(item ConversationItem) bool {
	return item.Kind == "SendActivity" || item.Kind == "Question"
}

// isInvokeAction returns true for workflow action kinds that invoke sub-agents.
func (t *Transformer) isInvokeAction(item ConversationItem) bool {
	return item.Kind == "InvokeAzureAgent" || item.Kind == "InvokeAgent"
}

// isSubAgentMessage returns true if a message was created by a sub-agent.
func (t *Transformer) isSubAgentMessage(item ConversationItem) bool {
	if item.CreatedBy == nil {
		return false
	}
	_, hasAgent := item.CreatedBy["agent"]
	return hasAgent
}

// markMCPGroupProcessed marks all items in an MCP approval group as processed.
func (t *Transformer) markMCPGroupProcessed(approvalRequestID string, mcpApprovalGroups map[string][]ConversationItem, processedIDs map[string]bool) {
	if related, ok := mcpApprovalGroups[approvalRequestID]; ok {
		for i := range related {
			processedIDs[related[i].ID] = true
		}
	}
}

// groupByApprovalRequestID groups MCP items by their approval_request_id.
func (t *Transformer) groupByApprovalRequestID(items []ConversationItem) map[string][]ConversationItem {
	groups := make(map[string][]ConversationItem)

	for i := range items {
		if items[i].ApprovalRequestID != "" {
			groups[items[i].ApprovalRequestID] = append(groups[items[i].ApprovalRequestID], items[i])
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

// hasApprovalRequest checks if there's an approval request item with the given ID.
func (t *Transformer) hasApprovalRequest(items []ConversationItem, approvalRequestID string) bool {
	for i := range items {
		if items[i].Type == "mcp_approval_request" && items[i].ID == approvalRequestID {
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
	switch item.Role {
	case "user":
		name = "User Message"
	case "assistant":
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

	inputText := RawMessageToString(approvalRequest.Arguments)
	outputText := ""
	if mcpCall != nil && len(mcpCall.Output) > 0 {
		outputText = RawMessageToString(mcpCall.Output)
	}

	status := models.NodeStatusCompleted
	if mcpResponse != nil && mcpResponse.Approve != nil && !*mcpResponse.Approve {
		status = models.NodeStatusCanceled
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
				Text: RawMessageToString(item.Arguments),
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
			status = models.NodeStatusCanceled
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
				Text: RawMessageToString(item.Arguments),
				Metadata: map[string]interface{}{
					"server_label": item.ServerLabel,
					"tool_name":    item.Name,
				},
			},
			Output: &models.NodeDataIO{
				Text: RawMessageToString(item.Output),
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

// markFunctionCallOutputProcessed marks the matching function_call_output as processed.
func (t *Transformer) markFunctionCallOutputProcessed(
	fcItem ConversationItem,
	functionCallOutputs map[string]*ConversationItem,
	processedIDs map[string]bool,
) {
	if fcItem.CallID != "" {
		if fco, ok := functionCallOutputs[fcItem.CallID]; ok {
			processedIDs[fco.ID] = true
		}
	}
}

// groupFunctionCallOutputsByCallID maps function_call_output / remote_function_call_output items by their call_id.
func (t *Transformer) groupFunctionCallOutputsByCallID(items []ConversationItem) map[string]*ConversationItem {
	outputs := make(map[string]*ConversationItem)

	for i := range items {
		if isFunctionCallOutputType(items[i].Type) && items[i].CallID != "" {
			item := items[i]
			outputs[items[i].CallID] = &item
		}
	}

	return outputs
}

// groupFunctionCallsByResponseID groups function_call / remote_function_call items by their response_id.
// This is used to nest function calls under the assistant message that triggered them.
func (t *Transformer) groupFunctionCallsByResponseID(items []ConversationItem) map[string][]ConversationItem {
	groups := make(map[string][]ConversationItem)

	for i := range items {
		if isFunctionCallType(items[i].Type) {
			if respID := t.extractResponseID(items[i]); respID != "" {
				groups[respID] = append(groups[respID], items[i])
			}
		}
	}

	return groups
}

// isMCPItemType returns true if the item type is an MCP item that can be nested.
func isMCPItemType(itemType string) bool {
	return itemType == "mcp_list_tools" || itemType == "mcp_call" || itemType == "mcp_approval_request"
}

// groupMCPItemsByResponseID groups MCP items (mcp_list_tools, mcp_call, mcp_approval_request)
// by their response_id. This is used to nest MCP items under the assistant message that triggered them.
func (t *Transformer) groupMCPItemsByResponseID(items []ConversationItem) map[string][]ConversationItem {
	groups := make(map[string][]ConversationItem)

	for i := range items {
		if isMCPItemType(items[i].Type) {
			if respID := t.extractResponseID(items[i]); respID != "" {
				groups[respID] = append(groups[respID], items[i])
			}
		}
	}

	return groups
}

// transformMCPItemForNesting transforms an MCP item into a TraceNode for nesting under an assistant message.
func (t *Transformer) transformMCPItemForNesting(
	item ConversationItem,
	mcpApprovalGroups map[string][]ConversationItem,
	createdBy string,
) models.TraceNode {
	switch item.Type {
	case "mcp_list_tools":
		return t.transformMCPListTools(item, createdBy)
	case "mcp_approval_request":
		return t.transformMCPGroup(item, mcpApprovalGroups, createdBy)
	case "mcp_call":
		return t.transformMCPCall(item, createdBy)
	default:
		return t.transformUnknown(item, createdBy)
	}
}

// transformFunctionCall transforms a function_call item and its matching function_call_output into a TraceNode.
func (t *Transformer) transformFunctionCall(
	item ConversationItem,
	functionCallOutputs map[string]*ConversationItem,
	createdBy string,
) models.TraceNode {
	now := time.Now().UTC()

	name := "Function Call"
	if item.Name != "" {
		name = item.Name
	}

	inputText := RawMessageToString(item.Arguments)
	outputText := ""
	if item.CallID != "" {
		if fco, ok := functionCallOutputs[item.CallID]; ok {
			outputText = RawMessageToString(fco.Output)
		}
	}

	node := models.TraceNode{
		ID:          "node_" + uuid.New().String(),
		Name:        name,
		Type:        models.NodeTypeTool,
		ReferenceID: item.ID,
		Status:      t.mapStatus(item.Status),
		Data: &models.NodeData{
			Input: &models.NodeDataIO{
				Text: inputText,
				Metadata: map[string]interface{}{
					"tool_name": item.Name,
					"call_id":   item.CallID,
				},
			},
			Output: &models.NodeDataIO{
				Text: outputText,
			},
		},
		Metadata:  t.buildFunctionCallMetadata(item),
		Nodes:     []models.TraceNode{},
		Logs:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: createdBy,
		UpdatedBy: createdBy,
	}

	return node
}

// transformFunctionCallOutput transforms a standalone function_call_output item into a TraceNode.
func (t *Transformer) transformFunctionCallOutput(item ConversationItem, createdBy string) models.TraceNode {
	now := time.Now().UTC()

	return models.TraceNode{
		ID:          "node_" + uuid.New().String(),
		Name:        "Function Call Output",
		Type:        models.NodeTypeTool,
		ReferenceID: item.ID,
		Status:      models.NodeStatusCompleted,
		Data: &models.NodeData{
			Output: &models.NodeDataIO{
				Text: RawMessageToString(item.Output),
				Metadata: map[string]interface{}{
					"call_id": item.CallID,
				},
			},
		},
		Metadata: map[string]interface{}{
			"partition_key": item.PartitionKey,
			"call_id":       item.CallID,
		},
		Nodes:     []models.TraceNode{},
		Logs:      []string{},
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: createdBy,
		UpdatedBy: createdBy,
	}
}

// buildFunctionCallMetadata builds metadata for a function_call node.
func (t *Transformer) buildFunctionCallMetadata(item ConversationItem) map[string]interface{} {
	metadata := map[string]interface{}{
		"partition_key": item.PartitionKey,
		"call_id":       item.CallID,
		"tool_name":     item.Name,
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
	case "cancelled": //nolint:misspell // value must stay "cancelled" for external API compatibility
		return models.NodeStatusCanceled
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
