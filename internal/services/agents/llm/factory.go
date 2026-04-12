package llm

import "fmt"

// NewStreamingClient creates a new streaming LLM client based on the provider.
func NewStreamingClient(provider string, config, credentialSecret map[string]interface{}) (StreamingClient, error) {
	apiKey := ""
	if credentialSecret != nil {
		if key, ok := credentialSecret["api_key"].(string); ok {
			apiKey = key
		}
	}

	switch provider {
	case "AZURE_OPENAI":
		return newAzureOpenAIStreamClient(config, apiKey)
	case "OPENAI":
		return newOpenAICompatibleStreamClient(config, apiKey, "https://api.openai.com", "Authorization", "Bearer "+apiKey)
	case "ANTHROPIC":
		return newAnthropicStreamClient(config, apiKey)
	case "GOOGLE_GENAI":
		return newGoogleGenAIStreamClient(config, apiKey)
	case "OLLAMA":
		return newOllamaStreamClient(config)
	case "MISTRAL":
		return newOpenAICompatibleStreamClient(config, apiKey, "https://api.mistral.ai", "Authorization", "Bearer "+apiKey)
	case "GROQ":
		return newOpenAICompatibleStreamClient(config, apiKey, "https://api.groq.com/openai", "Authorization", "Bearer "+apiKey)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", provider)
	}
}
