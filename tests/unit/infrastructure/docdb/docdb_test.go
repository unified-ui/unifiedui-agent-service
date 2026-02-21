// Package docdb_test provides unit tests for docdb core types and interfaces.
package docdb_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/core/docdb"
	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/tests/mocks"
)

// =============================================================================
// Type Constants Tests
// =============================================================================

func TestDocDBType_Constants(t *testing.T) {
	assert.Equal(t, docdb.Type("mongodb"), docdb.TypeMongoDB)
	assert.Equal(t, docdb.Type("cosmosdb"), docdb.TypeCosmosDB)
}

func TestSortOrder_Constants(t *testing.T) {
	assert.Equal(t, docdb.SortOrder("asc"), docdb.SortOrderAsc)
	assert.Equal(t, docdb.SortOrder("desc"), docdb.SortOrderDesc)
}

func TestSortField_Constants(t *testing.T) {
	assert.Equal(t, docdb.SortField("createdAt"), docdb.SortFieldCreatedAt)
	assert.Equal(t, docdb.SortField("updatedAt"), docdb.SortFieldUpdatedAt)
}

// =============================================================================
// UpdateResult and DeleteResult Tests
// =============================================================================

func TestUpdateResult_Fields(t *testing.T) {
	result := &docdb.UpdateResult{
		MatchedCount:  5,
		ModifiedCount: 3,
		UpsertedCount: 1,
		UpsertedID:    "new-id-123",
	}

	assert.Equal(t, int64(5), result.MatchedCount)
	assert.Equal(t, int64(3), result.ModifiedCount)
	assert.Equal(t, int64(1), result.UpsertedCount)
	assert.Equal(t, "new-id-123", result.UpsertedID)
}

func TestDeleteResult_Fields(t *testing.T) {
	result := &docdb.DeleteResult{
		DeletedCount: 10,
	}

	assert.Equal(t, int64(10), result.DeletedCount)
}

func TestFindOptions_Fields(t *testing.T) {
	opts := &docdb.FindOptions{
		Limit: 100,
		Skip:  50,
		Sort:  map[string]int{"createdAt": -1},
	}

	assert.Equal(t, int64(100), opts.Limit)
	assert.Equal(t, int64(50), opts.Skip)
	assert.NotNil(t, opts.Sort)
}

// =============================================================================
// ListMessagesOptions Tests
// =============================================================================

func TestListMessagesOptions_AllFields(t *testing.T) {
	opts := &docdb.ListMessagesOptions{
		ConversationID: "conv-123",
		TenantID:       "tenant-456",
		Type:           models.MessageTypeUser,
		Limit:          50,
		Skip:           10,
		OrderBy:        docdb.SortOrderDesc,
	}

	assert.Equal(t, "conv-123", opts.ConversationID)
	assert.Equal(t, "tenant-456", opts.TenantID)
	assert.Equal(t, models.MessageTypeUser, opts.Type)
	assert.Equal(t, int64(50), opts.Limit)
	assert.Equal(t, int64(10), opts.Skip)
	assert.Equal(t, docdb.SortOrderDesc, opts.OrderBy)
}

func TestListMessagesOptions_ZeroValues(t *testing.T) {
	opts := &docdb.ListMessagesOptions{}

	assert.Empty(t, opts.ConversationID)
	assert.Empty(t, opts.TenantID)
	assert.Empty(t, opts.Type)
	assert.Equal(t, int64(0), opts.Limit)
	assert.Equal(t, int64(0), opts.Skip)
	assert.Empty(t, opts.OrderBy)
}

// =============================================================================
// DeleteMessagesOptions Tests
// =============================================================================

func TestDeleteMessagesOptions_ByMessageID(t *testing.T) {
	opts := &docdb.DeleteMessagesOptions{
		MessageID: "msg-123",
		TenantID:  "tenant-456",
	}

	assert.Equal(t, "msg-123", opts.MessageID)
	assert.Equal(t, "tenant-456", opts.TenantID)
	assert.Empty(t, opts.ConversationID)
}

func TestDeleteMessagesOptions_ByConversation(t *testing.T) {
	opts := &docdb.DeleteMessagesOptions{
		ConversationID: "conv-789",
		TenantID:       "tenant-456",
	}

	assert.Equal(t, "conv-789", opts.ConversationID)
	assert.Equal(t, "tenant-456", opts.TenantID)
	assert.Empty(t, opts.MessageID)
}

// =============================================================================
// Reaction Options Tests
// =============================================================================

func TestUpsertReactionOptions_AllFields(t *testing.T) {
	opts := &docdb.UpsertReactionOptions{
		TenantID:       "tenant-123",
		ConversationID: "conv-456",
		MessageID:      "msg-789",
		UserID:         "user-101",
	}

	assert.Equal(t, "tenant-123", opts.TenantID)
	assert.Equal(t, "conv-456", opts.ConversationID)
	assert.Equal(t, "msg-789", opts.MessageID)
	assert.Equal(t, "user-101", opts.UserID)
}

func TestDeleteReactionOptions_AllFields(t *testing.T) {
	opts := &docdb.DeleteReactionOptions{
		TenantID:       "tenant-123",
		ConversationID: "conv-456",
		MessageID:      "msg-789",
		UserID:         "user-101",
	}

	assert.Equal(t, "tenant-123", opts.TenantID)
	assert.Equal(t, "conv-456", opts.ConversationID)
	assert.Equal(t, "msg-789", opts.MessageID)
	assert.Equal(t, "user-101", opts.UserID)
}

func TestListReactionsOptions_AllFields(t *testing.T) {
	opts := &docdb.ListReactionsOptions{
		TenantID:       "tenant-123",
		ConversationID: "conv-456",
		MessageID:      "msg-789",
	}

	assert.Equal(t, "tenant-123", opts.TenantID)
	assert.Equal(t, "conv-456", opts.ConversationID)
	assert.Equal(t, "msg-789", opts.MessageID)
}

// =============================================================================
// ListTracesOptions Tests
// =============================================================================

func TestListTracesOptions_AllFields(t *testing.T) {
	opts := &docdb.ListTracesOptions{
		TenantID:          "tenant-123",
		ChatAgentID:       "agent-456",
		ConversationID:    "conv-789",
		AutonomousAgentID: "auto-101",
		ContextType:       models.TraceContextConversation,
		Limit:             100,
		Skip:              25,
		OrderBy:           docdb.SortOrderDesc,
		SortBy:            docdb.SortFieldCreatedAt,
		Expand:            true,
	}

	assert.Equal(t, "tenant-123", opts.TenantID)
	assert.Equal(t, "agent-456", opts.ChatAgentID)
	assert.Equal(t, "conv-789", opts.ConversationID)
	assert.Equal(t, "auto-101", opts.AutonomousAgentID)
	assert.Equal(t, models.TraceContextConversation, opts.ContextType)
	assert.Equal(t, int64(100), opts.Limit)
	assert.Equal(t, int64(25), opts.Skip)
	assert.Equal(t, docdb.SortOrderDesc, opts.OrderBy)
	assert.Equal(t, docdb.SortFieldCreatedAt, opts.SortBy)
	assert.True(t, opts.Expand)
}

func TestListTracesOptions_MinimalFields(t *testing.T) {
	opts := &docdb.ListTracesOptions{
		TenantID: "tenant-123",
	}

	assert.Equal(t, "tenant-123", opts.TenantID)
	assert.Empty(t, opts.ChatAgentID)
	assert.Empty(t, opts.ConversationID)
	assert.Equal(t, int64(0), opts.Limit)
	assert.Nil(t, opts.CreatedAfter)
	assert.False(t, opts.Expand)
}

// =============================================================================
// Mock Interface Compliance Tests
// =============================================================================

func TestMockDocDBClient_ImplementsClientInterface(t *testing.T) {
	var _ docdb.Client = (*mocks.MockDocDBClient)(nil)

	client := mocks.NewMockDocDBClient()
	require.NotNil(t, client)

	assert.NotNil(t, client.Database())
	assert.NotNil(t, client.Messages())
	assert.NotNil(t, client.MessagesRaw())
	assert.NotNil(t, client.Reactions())
	assert.NotNil(t, client.Traces())
	assert.NotNil(t, client.TracesRaw())
}

func TestMockMessagesCollection_ImplementsInterface(t *testing.T) {
	var _ docdb.MessagesCollection = (*mocks.MockMessagesCollection)(nil)

	client := mocks.NewMockDocDBClient()
	messages := client.GetMessagesCollection()
	require.NotNil(t, messages)
}

func TestMockTracesCollection_ImplementsInterface(t *testing.T) {
	var _ docdb.TracesCollection = (*mocks.MockTracesCollection)(nil)

	client := mocks.NewMockDocDBClient()
	traces := client.GetTracesCollection()
	require.NotNil(t, traces)
}

func TestMockReactionsCollection_ImplementsInterface(t *testing.T) {
	var _ docdb.ReactionsCollection = (*mocks.MockReactionsCollection)(nil)

	client := mocks.NewMockDocDBClient()
	reactions := client.GetReactionsCollection()
	require.NotNil(t, reactions)
}

func TestMockCollection_ImplementsInterface(t *testing.T) {
	var _ docdb.Collection = (*mocks.MockCollection)(nil)
}

func TestMockDatabase_ImplementsInterface(t *testing.T) {
	var _ docdb.Database = (*mocks.MockDatabase)(nil)
}

func TestMockSingleResult_ImplementsInterface(t *testing.T) {
	var _ docdb.SingleResult = (*mocks.MockSingleResult)(nil)
}

func TestMockCursor_ImplementsInterface(t *testing.T) {
	var _ docdb.Cursor = (*mocks.MockCursor)(nil)
}

// =============================================================================
// Mock Client Helper Methods Tests
// =============================================================================

func TestMockDocDBClient_GetCollectionHelpers(t *testing.T) {
	client := mocks.NewMockDocDBClient()

	assert.NotNil(t, client.GetMessagesCollection())
	assert.NotNil(t, client.GetTracesCollection())
	assert.NotNil(t, client.GetTracesRawCollection())
	assert.NotNil(t, client.GetReactionsCollection())
}

func TestMockDocDBClient_Ping(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	ctx := context.Background()

	client.On("Ping", ctx).Return(nil)

	err := client.Ping(ctx)

	assert.NoError(t, err)
	client.AssertExpectations(t)
}

func TestMockDocDBClient_Close(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	ctx := context.Background()

	client.On("Close", ctx).Return(nil)

	err := client.Close(ctx)

	assert.NoError(t, err)
	client.AssertExpectations(t)
}

func TestMockDocDBClient_EnsureIndexes(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	ctx := context.Background()

	client.On("EnsureIndexes", ctx).Return(nil)

	err := client.EnsureIndexes(ctx)

	assert.NoError(t, err)
	client.AssertExpectations(t)
}

// =============================================================================
// Mock Messages Collection Tests
// =============================================================================

func TestMockMessagesCollection_Add(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	messages := client.GetMessagesCollection()
	ctx := context.Background()

	msg := &models.Message{
		ID:      "msg-123",
		Type:    models.MessageTypeUser,
		Content: "Hello",
	}

	messages.On("Add", ctx, msg).Return(nil)

	err := messages.Add(ctx, msg)

	assert.NoError(t, err)
	messages.AssertExpectations(t)
}

func TestMockMessagesCollection_Get(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	messages := client.GetMessagesCollection()
	ctx := context.Background()

	expected := &models.Message{
		ID:      "msg-123",
		Content: "Hello",
	}

	messages.On("Get", ctx, "msg-123").Return(expected, nil)

	result, err := messages.Get(ctx, "msg-123")

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	messages.AssertExpectations(t)
}

func TestMockMessagesCollection_Get_NotFound(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	messages := client.GetMessagesCollection()
	ctx := context.Background()

	messages.On("Get", ctx, "non-existent").Return(nil, nil)

	result, err := messages.Get(ctx, "non-existent")

	assert.NoError(t, err)
	assert.Nil(t, result)
	messages.AssertExpectations(t)
}

func TestMockMessagesCollection_List(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	messages := client.GetMessagesCollection()
	ctx := context.Background()

	opts := &docdb.ListMessagesOptions{
		ConversationID: "conv-123",
		TenantID:       "tenant-456",
		Limit:          10,
	}

	expected := []*models.Message{
		{ID: "msg-1", Content: "First"},
		{ID: "msg-2", Content: "Second"},
	}

	messages.On("List", ctx, opts).Return(expected, nil)

	result, err := messages.List(ctx, opts)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	messages.AssertExpectations(t)
}

func TestMockMessagesCollection_Update(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	messages := client.GetMessagesCollection()
	ctx := context.Background()

	msg := &models.Message{
		ID:      "msg-123",
		Content: "Updated content",
	}

	messages.On("Update", ctx, msg).Return(nil)

	err := messages.Update(ctx, msg)

	assert.NoError(t, err)
	messages.AssertExpectations(t)
}

func TestMockMessagesCollection_Delete(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	messages := client.GetMessagesCollection()
	ctx := context.Background()

	opts := &docdb.DeleteMessagesOptions{
		MessageID: "msg-123",
		TenantID:  "tenant-456",
	}

	messages.On("Delete", ctx, opts).Return(int64(1), nil)

	count, err := messages.Delete(ctx, opts)

	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
	messages.AssertExpectations(t)
}

func TestMockMessagesCollection_CountByConversation(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	messages := client.GetMessagesCollection()
	ctx := context.Background()

	messages.On("CountByConversation", ctx, "tenant-123", "conv-456").Return(int64(15), nil)

	count, err := messages.CountByConversation(ctx, "tenant-123", "conv-456")

	assert.NoError(t, err)
	assert.Equal(t, int64(15), count)
	messages.AssertExpectations(t)
}

func TestMockMessagesCollection_EnsureIndexes(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	messages := client.GetMessagesCollection()
	ctx := context.Background()

	messages.On("EnsureIndexes", ctx).Return(nil)

	err := messages.EnsureIndexes(ctx)

	assert.NoError(t, err)
	messages.AssertExpectations(t)
}

// =============================================================================
// Mock Traces Collection Tests
// =============================================================================

func TestMockTracesCollection_Create(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	traces := client.GetTracesCollection()
	ctx := context.Background()

	trace := &models.Trace{
		ID:       "trace-123",
		TenantID: "tenant-456",
	}

	traces.On("Create", ctx, trace).Return(nil)

	err := traces.Create(ctx, trace)

	assert.NoError(t, err)
	traces.AssertExpectations(t)
}

func TestMockTracesCollection_Get(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	traces := client.GetTracesCollection()
	ctx := context.Background()

	expected := &models.Trace{
		ID:       "trace-123",
		TenantID: "tenant-456",
	}

	traces.On("Get", ctx, "trace-123").Return(expected, nil)

	result, err := traces.Get(ctx, "trace-123")

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	traces.AssertExpectations(t)
}

func TestMockTracesCollection_GetByConversation(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	traces := client.GetTracesCollection()
	ctx := context.Background()

	expected := &models.Trace{
		ID:             "trace-123",
		TenantID:       "tenant-456",
		ConversationID: "conv-789",
	}

	traces.On("GetByConversation", ctx, "tenant-456", "conv-789").Return(expected, nil)

	result, err := traces.GetByConversation(ctx, "tenant-456", "conv-789")

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	traces.AssertExpectations(t)
}

func TestMockTracesCollection_GetByReferenceID(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	traces := client.GetTracesCollection()
	ctx := context.Background()

	expected := &models.Trace{
		ID:          "trace-123",
		TenantID:    "tenant-456",
		ReferenceID: "ext-ref-789",
	}

	traces.On("GetByReferenceID", ctx, "tenant-456", "ext-ref-789").Return(expected, nil)

	result, err := traces.GetByReferenceID(ctx, "tenant-456", "ext-ref-789")

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	traces.AssertExpectations(t)
}

func TestMockTracesCollection_List(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	traces := client.GetTracesCollection()
	ctx := context.Background()

	opts := &docdb.ListTracesOptions{
		TenantID: "tenant-123",
		Limit:    10,
	}

	expected := []*models.Trace{
		{ID: "trace-1"},
		{ID: "trace-2"},
	}

	traces.On("List", ctx, opts).Return(expected, nil)

	result, err := traces.List(ctx, opts)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	traces.AssertExpectations(t)
}

func TestMockTracesCollection_Count(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	traces := client.GetTracesCollection()
	ctx := context.Background()

	opts := &docdb.ListTracesOptions{
		TenantID: "tenant-123",
	}

	traces.On("Count", ctx, opts).Return(int64(42), nil)

	count, err := traces.Count(ctx, opts)

	assert.NoError(t, err)
	assert.Equal(t, int64(42), count)
	traces.AssertExpectations(t)
}

func TestMockTracesCollection_Update(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	traces := client.GetTracesCollection()
	ctx := context.Background()

	trace := &models.Trace{
		ID:       "trace-123",
		TenantID: "tenant-456",
	}

	traces.On("Update", ctx, trace).Return(nil)

	err := traces.Update(ctx, trace)

	assert.NoError(t, err)
	traces.AssertExpectations(t)
}

func TestMockTracesCollection_AddNodes(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	traces := client.GetTracesCollection()
	ctx := context.Background()

	nodes := []models.TraceNode{
		{ID: "node-1", Name: "Node 1"},
		{ID: "node-2", Name: "Node 2"},
	}

	traces.On("AddNodes", ctx, "trace-123", nodes).Return(nil)

	err := traces.AddNodes(ctx, "trace-123", nodes)

	assert.NoError(t, err)
	traces.AssertExpectations(t)
}

func TestMockTracesCollection_AddLogs(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	traces := client.GetTracesCollection()
	ctx := context.Background()

	logs := []string{"Log entry 1", "Log entry 2"}

	traces.On("AddLogs", ctx, "trace-123", logs).Return(nil)

	err := traces.AddLogs(ctx, "trace-123", logs)

	assert.NoError(t, err)
	traces.AssertExpectations(t)
}

func TestMockTracesCollection_Delete(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	traces := client.GetTracesCollection()
	ctx := context.Background()

	traces.On("Delete", ctx, "trace-123").Return(nil)

	err := traces.Delete(ctx, "trace-123")

	assert.NoError(t, err)
	traces.AssertExpectations(t)
}

func TestMockTracesCollection_DeleteByConversation(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	traces := client.GetTracesCollection()
	ctx := context.Background()

	traces.On("DeleteByConversation", ctx, "tenant-123", "conv-456").Return(nil)

	err := traces.DeleteByConversation(ctx, "tenant-123", "conv-456")

	assert.NoError(t, err)
	traces.AssertExpectations(t)
}

func TestMockTracesCollection_EnsureIndexes(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	traces := client.GetTracesCollection()
	ctx := context.Background()

	traces.On("EnsureIndexes", ctx).Return(nil)

	err := traces.EnsureIndexes(ctx)

	assert.NoError(t, err)
	traces.AssertExpectations(t)
}

// =============================================================================
// Mock Reactions Collection Tests
// =============================================================================

func TestMockReactionsCollection_Upsert(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	reactions := client.GetReactionsCollection()
	ctx := context.Background()

	reaction := &models.MessageReaction{
		TenantID:  "tenant-123",
		MessageID: "msg-456",
		UserID:    "user-789",
		Reaction:  models.ReactionThumbsUp,
	}

	reactions.On("Upsert", ctx, reaction).Return(nil)

	err := reactions.Upsert(ctx, reaction)

	assert.NoError(t, err)
	reactions.AssertExpectations(t)
}

func TestMockReactionsCollection_Get(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	reactions := client.GetReactionsCollection()
	ctx := context.Background()

	opts := &docdb.UpsertReactionOptions{
		TenantID:  "tenant-123",
		MessageID: "msg-456",
		UserID:    "user-789",
	}

	expected := &models.MessageReaction{
		TenantID:  "tenant-123",
		MessageID: "msg-456",
		UserID:    "user-789",
		Reaction:  models.ReactionThumbsUp,
	}

	reactions.On("Get", ctx, opts).Return(expected, nil)

	result, err := reactions.Get(ctx, opts)

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	reactions.AssertExpectations(t)
}

func TestMockReactionsCollection_ListByMessage(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	reactions := client.GetReactionsCollection()
	ctx := context.Background()

	opts := &docdb.ListReactionsOptions{
		TenantID:  "tenant-123",
		MessageID: "msg-456",
	}

	expected := []*models.MessageReaction{
		{UserID: "user-1", Reaction: models.ReactionThumbsUp},
		{UserID: "user-2", Reaction: models.ReactionThumbsDown},
	}

	reactions.On("ListByMessage", ctx, opts).Return(expected, nil)

	result, err := reactions.ListByMessage(ctx, opts)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	reactions.AssertExpectations(t)
}

func TestMockReactionsCollection_Delete(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	reactions := client.GetReactionsCollection()
	ctx := context.Background()

	opts := &docdb.DeleteReactionOptions{
		TenantID:  "tenant-123",
		MessageID: "msg-456",
		UserID:    "user-789",
	}

	reactions.On("Delete", ctx, opts).Return(nil)

	err := reactions.Delete(ctx, opts)

	assert.NoError(t, err)
	reactions.AssertExpectations(t)
}

func TestMockReactionsCollection_DeleteByConversation(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	reactions := client.GetReactionsCollection()
	ctx := context.Background()

	reactions.On("DeleteByConversation", ctx, "tenant-123", "conv-456").Return(nil)

	err := reactions.DeleteByConversation(ctx, "tenant-123", "conv-456")

	assert.NoError(t, err)
	reactions.AssertExpectations(t)
}

func TestMockReactionsCollection_EnsureIndexes(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	reactions := client.GetReactionsCollection()
	ctx := context.Background()

	reactions.On("EnsureIndexes", ctx).Return(nil)

	err := reactions.EnsureIndexes(ctx)

	assert.NoError(t, err)
	reactions.AssertExpectations(t)
}

// =============================================================================
// Messages Collection GetByUserMessageID Tests
// =============================================================================

func TestMockMessagesCollection_GetByUserMessageID(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	messages := client.GetMessagesCollection()
	ctx := context.Background()

	expected := &models.Message{
		ID:            "assistant-msg-123",
		Type:          models.MessageTypeAssistant,
		UserMessageID: "user-msg-456",
		Content:       "Assistant response",
	}

	messages.On("GetByUserMessageID", ctx, "user-msg-456").Return(expected, nil)

	result, err := messages.GetByUserMessageID(ctx, "user-msg-456")

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	assert.Equal(t, "user-msg-456", result.UserMessageID)
	messages.AssertExpectations(t)
}

func TestMockMessagesCollection_GetByUserMessageID_NotFound(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	messages := client.GetMessagesCollection()
	ctx := context.Background()

	messages.On("GetByUserMessageID", ctx, "non-existent").Return(nil, nil)

	result, err := messages.GetByUserMessageID(ctx, "non-existent")

	assert.NoError(t, err)
	assert.Nil(t, result)
	messages.AssertExpectations(t)
}

// =============================================================================
// Messages Collection ListChatHistory Tests
// =============================================================================

func TestMockMessagesCollection_ListChatHistory(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	messages := client.GetMessagesCollection()
	ctx := context.Background()

	opts := &docdb.ListMessagesOptions{
		ConversationID: "conv-123",
		TenantID:       "tenant-456",
		Limit:          30,
		OrderBy:        docdb.SortOrderAsc,
	}

	expected := []models.ChatHistoryEntry{
		{Role: models.MessageTypeUser, Content: "Hello"},
		{Role: models.MessageTypeAssistant, Content: "Hi there!"},
		{Role: models.MessageTypeUser, Content: "How are you?"},
	}

	messages.On("ListChatHistory", ctx, opts).Return(expected, nil)

	result, err := messages.ListChatHistory(ctx, opts)

	assert.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, models.MessageTypeUser, result[0].Role)
	messages.AssertExpectations(t)
}

// =============================================================================
// Traces Collection Autonomous Agent Tests
// =============================================================================

func TestMockTracesCollection_GetByAutonomousAgent(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	traces := client.GetTracesCollection()
	ctx := context.Background()

	expected := &models.Trace{
		ID:                "trace-123",
		TenantID:          "tenant-456",
		AutonomousAgentID: "auto-agent-789",
		ContextType:       models.TraceContextAutonomousAgent,
	}

	traces.On("GetByAutonomousAgent", ctx, "tenant-456", "auto-agent-789").Return(expected, nil)

	result, err := traces.GetByAutonomousAgent(ctx, "tenant-456", "auto-agent-789")

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	traces.AssertExpectations(t)
}

func TestMockTracesCollection_ListByConversation(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	traces := client.GetTracesCollection()
	ctx := context.Background()

	expected := []*models.Trace{
		{ID: "trace-1", ConversationID: "conv-123"},
		{ID: "trace-2", ConversationID: "conv-123"},
	}

	traces.On("ListByConversation", ctx, "tenant-456", "conv-123").Return(expected, nil)

	result, err := traces.ListByConversation(ctx, "tenant-456", "conv-123")

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	traces.AssertExpectations(t)
}

func TestMockTracesCollection_ListByAutonomousAgent(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	traces := client.GetTracesCollection()
	ctx := context.Background()

	expected := []*models.Trace{
		{ID: "trace-1", AutonomousAgentID: "auto-123"},
		{ID: "trace-2", AutonomousAgentID: "auto-123"},
	}

	traces.On("ListByAutonomousAgent", ctx, "tenant-456", "auto-123").Return(expected, nil)

	result, err := traces.ListByAutonomousAgent(ctx, "tenant-456", "auto-123")

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	traces.AssertExpectations(t)
}

func TestMockTracesCollection_DeleteByAutonomousAgent(t *testing.T) {
	client := mocks.NewMockDocDBClient()
	traces := client.GetTracesCollection()
	ctx := context.Background()

	traces.On("DeleteByAutonomousAgent", ctx, "tenant-123", "auto-456").Return(nil)

	err := traces.DeleteByAutonomousAgent(ctx, "tenant-123", "auto-456")

	assert.NoError(t, err)
	traces.AssertExpectations(t)
}
