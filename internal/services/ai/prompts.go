// Package ai provides the AI service for LLM interactions.
package ai

import "fmt"

const titleGenerationSystemPrompt = `You are a conversation title generator. Generate a concise, descriptive title (maximum 50 characters) for the following conversation. The title should capture the main topic. Return ONLY the title, nothing else. No quotes, no prefix.`

const descriptionGenerationWithExistingPrompt = `You are a technical writer. Improve the following raw description into a clear, professional, concise description (1-2 sentences, max 200 characters). Keep the original intent. Return ONLY the improved description, nothing else.

Entity type: %s
Entity name: %s
Raw description: %s`

const descriptionGenerationWithoutExistingPrompt = `You are a technical writer. Generate a clear, professional, concise description (1-2 sentences, max 200 characters) for the following entity. Return ONLY the description, nothing else.

Entity type: %s
Entity name: %s
Additional context: %s`

const traceErrorAnalysisPrompt = `You are a DevOps and AI agent debugging expert. Analyze the following trace error and provide:
1. Root cause analysis
2. Suggested fixes (actionable steps)
3. Prevention tips

Format your response in Markdown. Be concise but thorough.

Node: %s (type: %s)
Status: FAILED
Error: %s
Input (TOML):
%s
Output (TOML):
%s`

const traceSummarizationPrompt = `You are a trace analysis expert. Summarize the following agent execution trace.
Detail level: %s

For "short": 2-3 sentences covering status, total duration, and main flow.
For "medium": Include each major step, durations, and notable observations.
For "long": Detailed analysis of every node, data flow, performance, and recommendations.

Format in Markdown.

The trace is represented as a hierarchical tree. Indentation indicates parent-child relationships.

Trace nodes:
%s`

const testModelPrompt = "Reply with exactly: OK"

const traceChatSystemPrompt = `You are a trace analysis assistant. You have access to the full execution trace of an AI agent workflow provided as JSON. Answer the user's questions about the trace clearly and concisely. Use Markdown formatting.

The trace JSON includes root-level metadata (context type, reference info, logs, timestamps) and a hierarchical "nodes" array representing the execution tree.

Trace (JSON):
%s%s`

// BuildTitleGenerationMessages builds the messages for title generation.
func BuildTitleGenerationMessages(userMessage, assistantResponse string) []ChatMessage {
	snippet := assistantResponse
	if len(snippet) > 500 {
		snippet = snippet[:500]
	}

	return []ChatMessage{
		{Role: "system", Content: titleGenerationSystemPrompt},
		{Role: "user", Content: userMessage},
		{Role: "assistant", Content: snippet},
	}
}

// BuildDescriptionGenerationMessages builds the messages for description generation.
func BuildDescriptionGenerationMessages(entityType, entityName, existingDescription string, entityContext map[string]interface{}) []ChatMessage {
	var userContent string
	if existingDescription != "" {
		userContent = fmt.Sprintf(descriptionGenerationWithExistingPrompt, entityType, entityName, existingDescription)
	} else {
		contextStr := "{}"
		if entityContext != nil {
			contextStr = fmt.Sprintf("%v", entityContext)
		}
		userContent = fmt.Sprintf(descriptionGenerationWithoutExistingPrompt, entityType, entityName, contextStr)
	}

	return []ChatMessage{
		{Role: "user", Content: userContent},
	}
}

// BuildTraceAnalysisMessages builds the messages for trace error analysis.
func BuildTraceAnalysisMessages(nodeName, nodeType, errorMsg, inputTOML, outputTOML string) []ChatMessage {
	userContent := fmt.Sprintf(traceErrorAnalysisPrompt, nodeName, nodeType, errorMsg, inputTOML, outputTOML)

	return []ChatMessage{
		{Role: "user", Content: userContent},
	}
}

// BuildTraceSummarizeMessages builds the messages for trace summarization.
func BuildTraceSummarizeMessages(detailLevel, nodesTOML string) []ChatMessage {
	userContent := fmt.Sprintf(traceSummarizationPrompt, detailLevel, nodesTOML)

	return []ChatMessage{
		{Role: "user", Content: userContent},
	}
}

// BuildTestModelMessages builds the messages for testing a model.
func BuildTestModelMessages() []ChatMessage {
	return []ChatMessage{
		{Role: "user", Content: testModelPrompt},
	}
}

// BuildTraceChatMessages builds the messages for trace chat conversation.
func BuildTraceChatMessages(traceJSON string, selectedNodeJSON string, history []ChatMessage, userMessage string) []ChatMessage {
	selectedNodeSection := ""
	if selectedNodeJSON != "" {
		selectedNodeSection = fmt.Sprintf("\n\nCurrently selected/focused node (JSON):\n%s", selectedNodeJSON)
	}

	systemContent := fmt.Sprintf(traceChatSystemPrompt, traceJSON, selectedNodeSection)

	messages := []ChatMessage{
		{Role: "system", Content: systemContent},
	}

	messages = append(messages, history...)

	messages = append(messages, ChatMessage{
		Role:    "user",
		Content: userMessage,
	})

	return messages
}
