// Package docdb provides the traces collection interface.
package docdb

import (
	"context"

	"github.com/unifiedui/agent-service/internal/domain/models"
)

// ListTracesOptions contains options for listing traces.
type ListTracesOptions struct {
	TenantID          string
	ApplicationID     string
	ConversationID    string
	AutonomousAgentID string
	ContextType       models.TraceContextType // Optional: filter by context type
	Limit             int64
	Skip              int64
	OrderBy           SortOrder // Order by createdAt
}

// TracesCollection defines the interface for trace collection operations.
type TracesCollection interface {
	// Create inserts a new trace.
	Create(ctx context.Context, trace *models.Trace) error

	// Get retrieves a trace by ID.
	Get(ctx context.Context, id string) (*models.Trace, error)

	// GetByConversation retrieves a trace by conversation ID.
	// Returns the single trace for a conversation (one-to-one relationship).
	GetByConversation(ctx context.Context, tenantID, conversationID string) (*models.Trace, error)

	// GetByAutonomousAgent retrieves a trace by autonomous agent ID.
	// Returns the single trace for an autonomous agent (one-to-one relationship).
	GetByAutonomousAgent(ctx context.Context, tenantID, autonomousAgentID string) (*models.Trace, error)

	// List retrieves traces with pagination and filtering.
	List(ctx context.Context, opts *ListTracesOptions) ([]*models.Trace, error)

	// Update replaces an existing trace completely.
	Update(ctx context.Context, trace *models.Trace) error

	// AddNodes appends nodes to an existing trace.
	AddNodes(ctx context.Context, id string, nodes []models.TraceNode) error

	// AddLogs appends logs to an existing trace.
	AddLogs(ctx context.Context, id string, logs []string) error

	// Delete removes a trace by ID.
	Delete(ctx context.Context, id string) error

	// DeleteByConversation removes the trace for a conversation.
	DeleteByConversation(ctx context.Context, tenantID, conversationID string) error

	// DeleteByAutonomousAgent removes the trace for an autonomous agent.
	DeleteByAutonomousAgent(ctx context.Context, tenantID, autonomousAgentID string) error

	// EnsureIndexes creates necessary indexes for the collection.
	EnsureIndexes(ctx context.Context) error
}
