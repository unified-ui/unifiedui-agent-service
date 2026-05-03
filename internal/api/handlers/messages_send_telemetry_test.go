package handlers

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/unifiedui/agent-service/internal/api/middleware"
	"github.com/unifiedui/agent-service/internal/domain/models"
	"github.com/unifiedui/agent-service/internal/services/platform"
	"github.com/unifiedui/agent-service/internal/services/telemetry"
)

type captureSink struct {
	mu     sync.Mutex
	events []telemetry.MetricEvent
}

func (s *captureSink) Send(_ context.Context, batch []telemetry.MetricEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, batch...)
	return nil
}

func (s *captureSink) snapshot() []telemetry.MetricEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]telemetry.MetricEvent, len(s.events))
	copy(out, s.events)
	return out
}

func newTestEmitter(t *testing.T, sink telemetry.Sink) *telemetry.Emitter {
	t.Helper()
	emitter := telemetry.NewEmitter(sink, telemetry.Config{
		BufferSize:     16,
		FlushInterval:  20 * time.Millisecond,
		BatchSize:      1,
		MaxRetries:     1,
		InitialBackoff: time.Millisecond,
	})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		emitter.Stop()
		cancel()
	})
	emitter.Start(ctx)
	return emitter
}

func TestEmitSendMetric_Success(t *testing.T) {
	sink := &captureSink{}
	h := &MessagesHandler{}
	h.WithTelemetry(newTestEmitter(t, sink))

	tenantCtx := &middleware.TenantContext{TenantID: "t1", UserID: "u1"}
	agentConfig := &platform.AgentConfig{Type: platform.AgentTypeLLM}
	msg := &models.Message{
		ID:             "m1",
		ConversationID: "c1",
		ChatAgentID:    "ca1",
		Status:         models.MessageStatusSuccess,
		Metadata: &models.AssistantMetadata{
			Model:        "gpt-4o",
			TokensInput:  12,
			TokensOutput: 34,
			LatencyMs:    100,
			AgentType:    "llm",
		},
	}
	h.emitSendMetric(tenantCtx, agentConfig, msg, time.Now().Add(-50*time.Millisecond))

	assert.Eventually(t, func() bool {
		return len(sink.snapshot()) == 1
	}, 500*time.Millisecond, 5*time.Millisecond)

	ev := sink.snapshot()[0]
	assert.Equal(t, "t1", ev.TenantID)
	assert.Equal(t, "u1", ev.UserID)
	assert.Equal(t, "m1", ev.MessageID)
	assert.Equal(t, "ca1", ev.ChatAgentID)
	assert.Equal(t, "c1", ev.ConversationID)
	assert.Equal(t, "gpt-4o", ev.Model)
	assert.Equal(t, 12, ev.TokensInput)
	assert.Equal(t, 34, ev.TokensOutput)
	assert.Equal(t, "success", ev.Status)
	assert.Equal(t, "llm", ev.AgentType)
	assert.Equal(t, string(platform.AgentTypeLLM), ev.Provider)
	assert.Equal(t, "", ev.ErrorCode)
	assert.GreaterOrEqual(t, ev.LatencyMs, 100)
}

func TestEmitSendMetric_Failed(t *testing.T) {
	sink := &captureSink{}
	h := &MessagesHandler{}
	h.WithTelemetry(newTestEmitter(t, sink))

	tenantCtx := &middleware.TenantContext{TenantID: "t2", UserID: "u2"}
	agentConfig := &platform.AgentConfig{Type: platform.AgentTypeFoundry}
	msg := &models.Message{
		ID:             "m2",
		ConversationID: "c2",
		ChatAgentID:    "ca2",
		Status:         models.MessageStatusFailed,
	}
	h.emitSendMetric(tenantCtx, agentConfig, msg, time.Now().Add(-10*time.Millisecond))

	assert.Eventually(t, func() bool {
		return len(sink.snapshot()) == 1
	}, 500*time.Millisecond, 5*time.Millisecond)

	ev := sink.snapshot()[0]
	assert.Equal(t, "failed", ev.Status)
	assert.Equal(t, "STREAM_ERROR", ev.ErrorCode)
	assert.Equal(t, string(platform.AgentTypeFoundry), ev.Provider)
}

func TestEmitSendMetric_NoTelemetryNoOp(_ *testing.T) {
	h := &MessagesHandler{}
	tenantCtx := &middleware.TenantContext{TenantID: "t3", UserID: "u3"}
	msg := &models.Message{ID: "m3", Status: models.MessageStatusSuccess}
	h.emitSendMetric(tenantCtx, nil, msg, time.Now())
}
