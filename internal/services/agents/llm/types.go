// Package llm provides streaming LLM client implementations for direct model chat.
package llm

// StreamReader allows reading stream chunks one at a time from an LLM provider.
type StreamReader interface {
	// Read returns the next text chunk from the stream.
	// Returns io.EOF when the stream is exhausted.
	Read() (string, error)

	// Close releases resources associated with the reader.
	Close() error
}

// StreamingClient defines the interface for streaming LLM provider interactions.
type StreamingClient interface {
	// StreamChatCompletion sends a streaming chat completion request.
	StreamChatCompletion(messages []ChatMessage, opts StreamOptions) (StreamReader, error)

	// Close releases resources held by the client.
	Close() error
}

// ChatMessage represents a message in a chat completion request.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// StreamOptions holds optional streaming parameters.
type StreamOptions struct {
	Temperature  *float64
	MaxTokens    *int
	SystemPrompt *string
}
