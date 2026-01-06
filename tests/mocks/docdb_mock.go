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
	tracesCollection      *MockTracesCollection
	tracesRawCollection   *MockCollection
	database              *MockDatabase
}

// NewMockDocDBClient creates a new MockDocDBClient.
func NewMockDocDBClient() *MockDocDBClient {
	return &MockDocDBClient{
		messagesCollection:    &MockMessagesCollection{},
		messagesRawCollection: &MockCollection{},
		tracesCollection:      &MockTracesCollection{},
		tracesRawCollection:   &MockCollection{},
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

// Traces returns the typed traces collection.
func (m *MockDocDBClient) Traces() docdb.TracesCollection {
	return m.tracesCollection
}

// TracesRaw returns the raw traces collection.
func (m *MockDocDBClient) TracesRaw() docdb.Collection {
	return m.tracesRawCollection
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
func (m *MockDocDBClient) GetTracesCollection() *MockTracesCollection {
	return m.tracesCollection
}

// GetTracesRawCollection returns the mock raw traces collection for setup.
func (m *MockDocDBClient) GetTracesRawCollection() *MockCollection {
	return m.tracesRawCollection
}

// MockTracesCollection is a mock implementation of docdb.TracesCollection.
type MockTracesCollection struct {
	mock.Mock
}

// Create creates a new trace.
func (m *MockTracesCollection) Create(ctx context.Context, trace *models.Trace) error {
	args := m.Called(ctx, trace)
	return args.Error(0)
}

// Get gets a trace by ID.
func (m *MockTracesCollection) Get(ctx context.Context, id string) (*models.Trace, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Trace), args.Error(1)
}

// GetByConversation gets a trace by conversation ID.
func (m *MockTracesCollection) GetByConversation(ctx context.Context, tenantID, conversationID string) (*models.Trace, error) {
	args := m.Called(ctx, tenantID, conversationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Trace), args.Error(1)
}

// GetByAutonomousAgent gets a trace by autonomous agent ID.
func (m *MockTracesCollection) GetByAutonomousAgent(ctx context.Context, tenantID, autonomousAgentID string) (*models.Trace, error) {
	args := m.Called(ctx, tenantID, autonomousAgentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Trace), args.Error(1)
}

// List lists traces.
func (m *MockTracesCollection) List(ctx context.Context, opts *docdb.ListTracesOptions) ([]*models.Trace, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Trace), args.Error(1)
}

// Update updates a trace.
func (m *MockTracesCollection) Update(ctx context.Context, trace *models.Trace) error {
	args := m.Called(ctx, trace)
	return args.Error(0)
}

// AddNodes adds nodes to a trace.
func (m *MockTracesCollection) AddNodes(ctx context.Context, id string, nodes []models.TraceNode) error {
	args := m.Called(ctx, id, nodes)
	return args.Error(0)
}

// AddLogs adds logs to a trace.
func (m *MockTracesCollection) AddLogs(ctx context.Context, id string, logs []interface{}) error {
	args := m.Called(ctx, id, logs)
	return args.Error(0)
}

// Delete deletes a trace.
func (m *MockTracesCollection) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// DeleteByConversation deletes a trace by conversation.
func (m *MockTracesCollection) DeleteByConversation(ctx context.Context, tenantID, conversationID string) error {
	args := m.Called(ctx, tenantID, conversationID)
	return args.Error(0)
}

// DeleteByAutonomousAgent deletes a trace by autonomous agent.
func (m *MockTracesCollection) DeleteByAutonomousAgent(ctx context.Context, tenantID, autonomousAgentID string) error {
	args := m.Called(ctx, tenantID, autonomousAgentID)
	return args.Error(0)
}

// EnsureIndexes creates indexes.
func (m *MockTracesCollection) EnsureIndexes(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockMessagesCollection is a mock implementation of docdb.MessagesCollection.
type MockMessagesCollection struct {
	mock.Mock
}

// Add adds a message.
func (m *MockMessagesCollection) Add(ctx context.Context, message *models.Message) error {
	args := m.Called(ctx, message)
	return args.Error(0)
}

// Get gets a message by ID.
func (m *MockMessagesCollection) Get(ctx context.Context, id string) (*models.Message, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Message), args.Error(1)
}

// GetByUserMessageID gets assistant message by user message ID.
func (m *MockMessagesCollection) GetByUserMessageID(ctx context.Context, userMessageID string) (*models.Message, error) {
	args := m.Called(ctx, userMessageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Message), args.Error(1)
}

// List lists messages.
func (m *MockMessagesCollection) List(ctx context.Context, opts *docdb.ListMessagesOptions) ([]*models.Message, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Message), args.Error(1)
}

// ListChatHistory lists chat history.
func (m *MockMessagesCollection) ListChatHistory(ctx context.Context, opts *docdb.ListMessagesOptions) ([]models.ChatHistoryEntry, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.ChatHistoryEntry), args.Error(1)
}

// Update updates a message.
func (m *MockMessagesCollection) Update(ctx context.Context, message *models.Message) error {
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
