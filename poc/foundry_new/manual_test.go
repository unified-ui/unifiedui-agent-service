//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/traceimport/foundry"
)

func main() {
	files := []string{
		"poc/foundry_new/resp_tracing_conv_c671246113e78fe700QvxJVpUm1OlmSlm4N81oS4QPxhQ2Og1b.json",
		"poc/foundry_new/resp_tracing5.json",
	}

	for _, file := range files {
		fmt.Printf("\n=== %s ===\n", file)
		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			continue
		}

		var resp foundry.ConversationItemsResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			fmt.Printf("Error parsing JSON: %v\n", err)
			continue
		}

		transformer := foundry.NewTransformer()
		nodes := transformer.Transform(resp.Data, "test-user")

		printNodes(nodes, 0)
	}
}

func printNodes(nodes []models.TraceNode, depth int) {
	indent := strings.Repeat("  ", depth)
	for _, n := range nodes {
		childCount := ""
		if len(n.Nodes) > 0 {
			childCount = fmt.Sprintf(" (%d children)", len(n.Nodes))
		}
		inputText := ""
		outputText := ""
		if n.Data != nil {
			if n.Data.Input != nil && n.Data.Input.Text != "" {
				t := n.Data.Input.Text
				if len(t) > 60 {
					t = t[:60] + "..."
				}
				inputText = fmt.Sprintf(" IN=%q", t)
			}
			if n.Data.Output != nil && n.Data.Output.Text != "" {
				t := n.Data.Output.Text
				if len(t) > 60 {
					t = t[:60] + "..."
				}
				outputText = fmt.Sprintf(" OUT=%q", t)
			}
		}
		fmt.Printf("%s├── [%s] %s [%s]%s%s%s\n", indent, n.Type, n.Name, n.Status, childCount, inputText, outputText)
		if len(n.Nodes) > 0 {
			printNodes(n.Nodes, depth+1)
		}
	}
}
