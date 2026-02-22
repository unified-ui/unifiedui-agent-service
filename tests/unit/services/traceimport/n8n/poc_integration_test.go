package n8n_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/domain/models"
	n8n "github.com/unifiedui/agent-service/internal/services/traceimport/n8n"
)

func pocFilePath(filename string) string {
	_, currentFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "..", "..", "poc", "n8n", "tracing", "tmp", filename)
}

func loadExecution(t *testing.T, filename string) *n8n.ExecutionResponse {
	t.Helper()
	path := pocFilePath(filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skipf("skipping: fixture file %s not found (POC local-only data)", filename)
	}
	data, err := os.ReadFile(path)
	require.NoError(t, err, "failed to read %s", filename)

	var exec n8n.ExecutionResponse
	err = json.Unmarshal(data, &exec)
	require.NoError(t, err, "failed to parse %s", filename)

	return &exec
}

func printTraceTree(t *testing.T, nodes []models.TraceNode) {
	t.Helper()
	for _, node := range nodes {
		printNodeTree(t, node, 0)
	}
}

func printNodeTree(t *testing.T, node models.TraceNode, indent int) {
	t.Helper()
	prefix := strings.Repeat("  ", indent)
	outputText := ""
	inputText := ""
	if node.Data != nil {
		if node.Data.Output != nil && node.Data.Output.Text != "" {
			outputText = node.Data.Output.Text
			if len(outputText) > 80 {
				outputText = outputText[:80] + "..."
			}
		}
		if node.Data.Input != nil && node.Data.Input.Text != "" {
			inputText = node.Data.Input.Text
			if len(inputText) > 80 {
				inputText = inputText[:80] + "..."
			}
		}
	}
	t.Logf("%s- %s [type=%s, status=%s]", prefix, node.Name, node.Type, node.Status)
	if inputText != "" {
		t.Logf("%s  input: %s", prefix, inputText)
	}
	if outputText != "" {
		t.Logf("%s  output: %s", prefix, outputText)
	}
	if node.Data != nil && node.Data.Input != nil && node.Data.Input.ExtraData != nil {
		t.Logf("%s  input_extra_keys: %v", prefix, extractMapKeys(node.Data.Input.ExtraData))
	}
	if node.Data != nil && node.Data.Output != nil && node.Data.Output.ExtraData != nil {
		t.Logf("%s  output_extra_keys: %v", prefix, extractMapKeys(node.Data.Output.ExtraData))
	}
	if len(node.Metadata) > 0 {
		if connType, ok := node.Metadata["connection_type"]; ok {
			t.Logf("%s  connection_type: %s", prefix, connType)
		}
	}
	for _, child := range node.Nodes {
		printNodeTree(t, child, indent+1)
	}
}

func extractMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func findNode(nodes []models.TraceNode, name string) *models.TraceNode {
	for i := range nodes {
		if nodes[i].Name == name {
			return &nodes[i]
		}
		if found := findNodeRecursive(nodes[i].Nodes, name); found != nil {
			return found
		}
	}
	return nil
}

func findNodeRecursive(nodes []models.TraceNode, name string) *models.TraceNode {
	for i := range nodes {
		if nodes[i].Name == name {
			return &nodes[i]
		}
		if found := findNodeRecursive(nodes[i].Nodes, name); found != nil {
			return found
		}
	}
	return nil
}

func TestPOC_Execution119_AIAgentWorkflow(t *testing.T) {
	exec := loadExecution(t, "execution_119.json")
	transformer := n8n.NewTransformer()
	nodes := transformer.TransformExecution(exec, "test-user")

	t.Log("=== execution_119: AI Agent Chat Response Workflow ===")
	printTraceTree(t, nodes)

	require.Equal(t, 3, len(nodes), "expected 3 top-level nodes (chatTrigger, agent, respondToWebhook)")

	chatTrigger := findNode(nodes, "When chat message received")
	require.NotNil(t, chatTrigger, "chatTrigger node should exist")
	assert.Equal(t, models.NodeTypeWorkflow, chatTrigger.Type)
	assert.Equal(t, models.NodeStatusCompleted, chatTrigger.Status)
	assert.NotNil(t, chatTrigger.Data, "chatTrigger should have data")

	agent := findNode(nodes, "AI Agent")
	require.NotNil(t, agent, "AI Agent node should exist")
	assert.Equal(t, models.NodeTypeAgent, agent.Type)
	assert.Equal(t, models.NodeStatusCompleted, agent.Status)
	assert.NotNil(t, agent.Data, "agent should have output data")
	if agent.Data != nil && agent.Data.Output != nil {
		assert.NotEmpty(t, agent.Data.Output.Text, "agent should have output text")
	}

	require.GreaterOrEqual(t, len(agent.Nodes), 1, "agent should have at least 1 child (LLM)")
	llmNode := findNode(agent.Nodes, "Azure OpenAI Chat Model")
	require.NotNil(t, llmNode, "LLM sub-node should be a child of agent")
	assert.Equal(t, models.NodeTypeLLM, llmNode.Type)
	assert.Equal(t, models.NodeStatusCompleted, llmNode.Status)
	assert.Equal(t, "ai_languageModel", llmNode.Metadata["connection_type"])

	if llmNode.Data != nil && llmNode.Data.Output != nil {
		t.Logf("LLM output text: %q", llmNode.Data.Output.Text)
	}
	if llmNode.Data != nil && llmNode.Data.Input != nil {
		t.Logf("LLM has input data with extra keys: %v", extractMapKeys(llmNode.Data.Input.ExtraData))
	}

	respondToWebhook := findNode(nodes, "Respond to Webhook")
	require.NotNil(t, respondToWebhook, "respondToWebhook node should exist")
	assert.Equal(t, models.NodeTypeWorkflow, respondToWebhook.Type)
}

func TestPOC_Execution1648_APIWorkflow(t *testing.T) {
	exec := loadExecution(t, "execution_1648.json")
	transformer := n8n.NewTransformer()
	nodes := transformer.TransformExecution(exec, "test-user")

	t.Log("=== execution_1648: GoogleSearchSites API Workflow ===")
	printTraceTree(t, nodes)

	require.Equal(t, 3, len(nodes), "expected 3 top-level nodes")

	trigger := findNode(nodes, "When clicking \u2018Execute workflow\u2019")
	require.NotNil(t, trigger)
	assert.Equal(t, models.NodeTypeWorkflow, trigger.Type)

	setNode := findNode(nodes, "Edit Fields")
	require.NotNil(t, setNode)
	assert.Equal(t, models.NodeTypeCustom, setNode.Type)

	httpNode := findNode(nodes, "HTTP Request")
	require.NotNil(t, httpNode)
	assert.Equal(t, models.NodeTypeHTTP, httpNode.Type)
	assert.NotNil(t, httpNode.Data, "httpRequest should have data")
}

func TestPOC_Execution1713_FormWorkflow(t *testing.T) {
	exec := loadExecution(t, "execution_1713.json")
	transformer := n8n.NewTransformer()
	nodes := transformer.TransformExecution(exec, "test-user")

	t.Log("=== execution_1713: Create Project Form Workflow ===")
	printTraceTree(t, nodes)

	require.Equal(t, 4, len(nodes), "expected 4 top-level nodes")

	formTrigger := findNode(nodes, "On form submission")
	require.NotNil(t, formTrigger)
	assert.Equal(t, models.NodeTypeWorkflow, formTrigger.Type)

	postgres := findNode(nodes, "Execute a SQL query")
	require.NotNil(t, postgres)
	assert.Equal(t, models.NodeTypeTool, postgres.Type)

	code := findNode(nodes, "Code in JavaScript")
	require.NotNil(t, code)
	assert.Equal(t, models.NodeTypeCode, code.Type)

	form := findNode(nodes, "Form")
	require.NotNil(t, form)
	assert.Equal(t, models.NodeTypeCustom, form.Type)
}

func TestPOC_Execution1715_DataProcessingWorkflow(t *testing.T) {
	exec := loadExecution(t, "execution_1715.json")
	transformer := n8n.NewTransformer()
	nodes := transformer.TransformExecution(exec, "test-user")

	t.Log("=== execution_1715: Kryo Data Processing Workflow ===")
	printTraceTree(t, nodes)

	require.Equal(t, 7, len(nodes), "expected 7 top-level nodes")

	trigger := findNode(nodes, "When clicking \u2018Execute workflow\u2019")
	require.NotNil(t, trigger)
	assert.Equal(t, models.NodeTypeWorkflow, trigger.Type)

	execCmd := findNode(nodes, "Execute Command")
	require.NotNil(t, execCmd)
	assert.Equal(t, models.NodeTypeTool, execCmd.Type)

	code := findNode(nodes, "Code in JavaScript3")
	require.NotNil(t, code)
	assert.Equal(t, models.NodeTypeCode, code.Type)

	merge := findNode(nodes, "Merge")
	require.NotNil(t, merge)
	assert.Equal(t, models.NodeTypeWorkflow, merge.Type)

	splitInBatches := findNode(nodes, "Loop Over Items")
	require.NotNil(t, splitInBatches)
	assert.Equal(t, models.NodeTypeLoop, splitInBatches.Type)

	readWriteFile := findNode(nodes, "Read/Write Files from Disk")
	require.NotNil(t, readWriteFile)
	assert.Equal(t, models.NodeTypeTool, readWriteFile.Type)
}

func TestPOC_Execution1716_ScrapingWorkflow(t *testing.T) {
	exec := loadExecution(t, "execution_1716.json")
	transformer := n8n.NewTransformer()
	nodes := transformer.TransformExecution(exec, "test-user")

	t.Log("=== execution_1716: Scraping Workflow ===")
	printTraceTree(t, nodes)

	require.Equal(t, 8, len(nodes), "expected 8 top-level nodes")

	trigger := findNode(nodes, "When clicking \u2018Execute workflow\u2019")
	require.NotNil(t, trigger)
	assert.Equal(t, models.NodeTypeWorkflow, trigger.Type)

	switchNode := findNode(nodes, "Switch")
	require.NotNil(t, switchNode)
	assert.Equal(t, models.NodeTypeConditional, switchNode.Type)

	htmlNode := findNode(nodes, "HTML")
	require.NotNil(t, htmlNode)
	assert.Equal(t, models.NodeTypeCustom, htmlNode.Type)

	httpNode := findNode(nodes, "HTTP Request")
	require.NotNil(t, httpNode)
	assert.Equal(t, models.NodeTypeHTTP, httpNode.Type)

	for _, node := range nodes {
		assert.Equal(t, models.NodeStatusCompleted, node.Status,
			fmt.Sprintf("node %s should be completed", node.Name))
	}
}

func TestPOC_AllExecutions_CommonAssertions(t *testing.T) {
	files := []string{
		"execution_119.json",
		"execution_1648.json",
		"execution_1713.json",
		"execution_1715.json",
		"execution_1716.json",
	}

	transformer := n8n.NewTransformer()

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			exec := loadExecution(t, file)
			nodes := transformer.TransformExecution(exec, "test-user")

			require.Greater(t, len(nodes), 0, "should produce at least one trace node")

			for _, node := range nodes {
				assertNodeValid(t, node, 0)
			}

			assertChronologicalOrder(t, nodes)
		})
	}
}

func assertNodeValid(t *testing.T, node models.TraceNode, depth int) {
	t.Helper()

	assert.NotEmpty(t, node.ID, "node should have an ID")
	assert.NotEmpty(t, node.Name, "node should have a name")
	assert.NotEmpty(t, string(node.Type), "node should have a type")
	assert.NotEmpty(t, string(node.Status), "node should have a status")
	assert.NotNil(t, node.StartAt, "node %s should have start time", node.Name)
	assert.NotNil(t, node.Metadata, "node %s should have metadata", node.Name)
	assert.Contains(t, node.Metadata, "n8n_node_type",
		"node %s metadata should contain n8n_node_type", node.Name)

	if node.StartAt != nil && node.EndAt != nil {
		assert.False(t, node.EndAt.Before(*node.StartAt),
			"node %s end time should not be before start time", node.Name)
	}

	assert.GreaterOrEqual(t, node.Duration, 0.0,
		"node %s duration should be non-negative", node.Name)

	for _, child := range node.Nodes {
		assertNodeValid(t, child, depth+1)
		if child.Metadata != nil {
			_, hasConnType := child.Metadata["connection_type"]
			assert.True(t, hasConnType,
				"child node %s should have connection_type metadata", child.Name)
		}
	}
}

func assertChronologicalOrder(t *testing.T, nodes []models.TraceNode) {
	t.Helper()
	for i := 1; i < len(nodes); i++ {
		if nodes[i-1].StartAt != nil && nodes[i].StartAt != nil {
			assert.False(t, nodes[i].StartAt.Before(*nodes[i-1].StartAt),
				"nodes should be in chronological order: %s (%v) before %s (%v)",
				nodes[i].Name, nodes[i].StartAt, nodes[i-1].Name, nodes[i-1].StartAt)
		}
	}
}
