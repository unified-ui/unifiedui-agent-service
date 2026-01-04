// Package mocks provides mock implementations for testing.
package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/unifiedui/agent-service/internal/core/docdb"
	"github.com/unifiedui/agent-service/internal/domain/models"
)

// MockCollection is a mock implementation of docdb.Collection.
type MockCollection struct {
	mock.Mock
}

// InsertOne inserts a single document.
func (m *MockCollection) InsertOne(ctx context.Context, document interface{}) (interface{}, error) {
	args := m.Called(ctx, document)
	return args.Get(0), args.Error(1)
}

// InsertMany inserts multiple documents.
func (m *MockCollection) InsertMany(ctx context.Context, documents []interface{}) ([]interface{}, error) {
	args := m.Called(ctx, documents)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]interface{}), args.Error(1)
}

// FindOne finds a single document.
func (m *MockCollection) FindOne(ctx context.Context, filter interface{}) docdb.SingleResult {
	args := m.Called(ctx, filter)
	return args.Get(0).(docdb.SingleResult)
}

// Find finds multiple documents.
func (m *MockCollection) Find(ctx context.Context, filter interface{}, opts *docdb.FindOptions) (docdb.Cursor, error) {
	args := m.Called(ctx, filter, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(docdb.Cursor), args.Error(1)
}

// UpdateOne updates a single document.
func (m *MockCollection) UpdateOne(ctx context.Context, filter interface{}, update interface{}) (*docdb.UpdateResult, error) {
	args := m.Called(ctx, filter, update)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*docdb.UpdateResult), args.Error(1)
}

// UpdateMany updates multiple documents.
func (m *MockCollection) UpdateMany(ctx context.Context, filter interface{}, update interface{}) (*docdb.UpdateResult, error) {
	args := m.Called(ctx, filter, update)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*docdb.UpdateResult), args.Error(1)
}

// DeleteOne deletes a single document.
func (m *MockCollection) DeleteOne(ctx context.Context, filter interface{}) (*docdb.DeleteResult, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*docdb.DeleteResult), args.Error(1)
}

// DeleteMany deletes multiple documents.
func (m *MockCollection) DeleteMany(ctx context.Context, filter interface{}) (*docdb.DeleteResult, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*docdb.DeleteResult), args.Error(1)
}

// CountDocuments counts documents matching the filter.
func (m *MockCollection) CountDocuments(ctx context.Context, filter interface{}) (int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(int64), args.Error(1)
}

// MockDatabase is a mock implementation of docdb.Database.
type MockDatabase struct {
	mock.Mock
}

// Collection returns a collection from the database.
func (m *MockDatabase) Collection(name string) docdb.Collection {
	args := m.Called(name)
	return args.Get(0).(docdb.Collection)
}

// ListCollectionNames lists all collection names.
func (m *MockDatabase) ListCollectionNames(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

// MockDocDBClient is a mock implementation of docdb.Client.
type MockDocDBClient struct {
	mock.Mock
	messagesCollection    *MockMessagesCollection
	messagesRawCollection *MockCollection
	tracesCollection      *MockCollection
	database              *MockDatabase
}

// NewMockDocDBClient creates a new MockDocDBClient.
func NewMockDocDBClient() *MockDocDBClient {
	return &MockDocDBClient{
		messagesCollection:    &MockMessagesCollection{},
		messagesRawCollection: &MockCollection{},
		tracesCollection:      &MockCollection{},
		database:              &MockDatabase{},
	}
}

// Database returns the database.
func (m *MockDocDBClient) Database() docdb.Database {
	return m.database
}

// Messages returns the typed messages collection.
func (m *MockDocDBClient) Messages() docdb.MessagesCollection {
	return m.messagesCollection
}

// MessagesRaw returns the raw messages collection.
func (m *MockDocDBClient) MessagesRaw() docdb.Collection {
	return m.messagesRawCollection
}

// Traces returns the traces collection.
func (m *MockDocDBClient) Traces() docdb.Collection {
	return m.tracesCollection
}

// Ping checks the database connection.
func (m *MockDocDBClient) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Close closes the database connection.
func (m *MockDocDBClient) Close(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// EnsureIndexes creates all necessary indexes.
func (m *MockDocDBClient) EnsureIndexes(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// GetMessagesCollection returns the mock messages collection for setup.
func (m *MockDocDBClient) GetMessagesCollection() *MockMessagesCollection {
	return m.messagesCollection
}

// GetTracesCollection returns the mock traces collection for setup.
func (m *MockDocDBClient) GetTracesCollection() *MockCollection {
	return m.tracesCollection
}

// MockMessagesCollection is a mock implementation of docdb.MessagesCollection.
type MockMessagesCollection struct {
	mock.Mock
}

// AddUserMessage adds a user message.
func (m *MockMessagesCollection) AddUserMessage(ctx context.Context, message *models.UserMessage) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

// AddAssistantMessage adds an assistant message.
func (m *MockMessagesCollection) AddAssistantMessage(ctx context.Context, message *models.AssistantMessage) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

// GetUserMessage gets a user message by ID.
func (m *MockMessagesCollection) GetUserMessage(ctx context.Context, id string) (*models.UserMessage, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserMessage), args.Error(1)
}

// GetAssistantMessage gets an assistant message by ID.
func (m *MockMessagesCollection) GetAssistantMessage(ctx context.Context, id string) (*models.AssistantMessage, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AssistantMessage), args.Error(1)
}

// GetAssistantMessageByUserMessageID gets assistant message by user message ID.
func (m *MockMessagesCollection) GetAssistantMessageByUserMessageID(ctx context.Context, userMessageID string) (*models.AssistantMessage, error) {
	args := m.Called(ctx, userMessageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.AssistantMessage), args.Error(1)
}

// ListUserMessages lists user messages.
func (m *MockMessagesCollection) ListUserMessages(ctx context.Context, opts *docdb.ListMessagesOptions) ([]*models.UserMessage, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.UserMessage), args.Error(1)
}

// ListAssistantMessages lists assistant messages.
func (m *MockMessagesCollection) ListAssistantMessages(ctx context.Context, opts *docdb.ListMessagesOptions) ([]*models.AssistantMessage, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.AssistantMessage), args.Error(1)
}

// ListChatHistory lists chat history.
func (m *MockMessagesCollection) ListChatHistory(ctx context.Context, opts *docdb.ListMessagesOptions) ([]models.ChatHistoryEntry, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.ChatHistoryEntry), args.Error(1)
}

// UpdateUserMessage updates a user message.
func (m *MockMessagesCollection) UpdateUserMessage(ctx context.Context, message *models.UserMessage) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

// UpdateAssistantMessage updates an assistant message.
func (m *MockMessagesCollection) UpdateAssistantMessage(ctx context.Context, message *models.AssistantMessage) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

// Delete deletes messages.
func (m *MockMessagesCollection) Delete(ctx context.Context, opts *docdb.DeleteMessagesOptions) (int64, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(int64), args.Error(1)
}

// CountByConversation counts messages by conversation.
func (m *MockMessagesCollection) CountByConversation(ctx context.Context, tenantID, conversationID string) (int64, error) {
	args := m.Called(ctx, tenantID, conversationID)
	return args.Get(0).(int64), args.Error(1)
}

// EnsureIndexes creates indexes.
func (m *MockMessagesCollection) EnsureIndexes(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockSingleResult is a mock implementation of docdb.SingleResult.
type MockSingleResult struct {
	mock.Mock
}

// Decode decodes the result.
func (m *MockSingleResult) Decode(v interface{}) error {
	args := m.Called(v)
	return args.Error(0)
}

// Err returns any error.
func (m *MockSingleResult) Err() error {
	args := m.Called()
	return args.Error(0)
}

// MockCursor is a mock implementation of docdb.Cursor.
type MockCursor struct {
	mock.Mock
}

// Next advances the cursor.
func (m *MockCursor) Next(ctx context.Context) bool {
	args := m.Called(ctx)
	return args.Bool(0)
}

// Decode decodes the current document.
func (m *MockCursor) Decode(v interface{}) error {
	args := m.Called(v)
	return args.Error(0)
}

// All decodes all documents.
func (m *MockCursor) All(ctx context.Context, results interface{}) error {
	args := m.Called(ctx, results)
	return args.Error(0)
}

// Err returns any cursor error.
func (m *MockCursor) Err() error {
	args := m.Called()
	return args.Error(0)
}

// Close closes the cursor.
func (m *MockCursor) Close(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}
