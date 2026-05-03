package telemetry_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/services/telemetry"
)

// recordingSink is a Sink mock that captures dispatched batches.
type recordingSink struct {
	mu      sync.Mutex
	batches [][]telemetry.MetricEvent
	failN   int32
	failErr error
}

func (s *recordingSink) Send(_ context.Context, batch []telemetry.MetricEvent) error {
	if atomic.LoadInt32(&s.failN) > 0 {
		atomic.AddInt32(&s.failN, -1)
		err := s.failErr
		if err == nil {
			err = errors.New("forced failure")
		}
		return err
	}
	cp := make([]telemetry.MetricEvent, len(batch))
	copy(cp, batch)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.batches = append(s.batches, cp)
	return nil
}

func (s *recordingSink) Batches() [][]telemetry.MetricEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([][]telemetry.MetricEvent, len(s.batches))
	copy(out, s.batches)
	return out
}

func waitForBatches(t *testing.T, sink *recordingSink, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(sink.Batches()) >= want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected at least %d batch(es), got %d", want, len(sink.Batches()))
}

func TestEmitter_FlushOnBatchSize(t *testing.T) {
	sink := &recordingSink{}
	em := telemetry.NewEmitter(sink, telemetry.Config{
		BufferSize:    10,
		BatchSize:     3,
		FlushInterval: time.Hour,
	})
	em.Start(context.Background())
	defer em.Stop()

	for i := 0; i < 3; i++ {
		require.True(t, em.Emit(telemetry.MetricEvent{TenantID: "t", MessageID: "m"}))
	}
	waitForBatches(t, sink, 1)
	assert.Len(t, sink.Batches()[0], 3)
}

func TestEmitter_FlushOnTick(t *testing.T) {
	sink := &recordingSink{}
	em := telemetry.NewEmitter(sink, telemetry.Config{
		BufferSize:    10,
		BatchSize:     50,
		FlushInterval: 50 * time.Millisecond,
	})
	em.Start(context.Background())
	defer em.Stop()

	em.Emit(telemetry.MetricEvent{MessageID: "m1"})
	em.Emit(telemetry.MetricEvent{MessageID: "m2"})
	waitForBatches(t, sink, 1)
	assert.Len(t, sink.Batches()[0], 2)
}

func TestEmitter_DropOnFullBuffer(t *testing.T) {
	blocker := make(chan struct{})
	sink := blockingSink(blocker)

	em := telemetry.NewEmitter(sink, telemetry.Config{
		BufferSize:    2,
		BatchSize:     1,
		FlushInterval: time.Hour,
	})
	em.Start(context.Background())
	defer func() {
		close(blocker)
		em.Stop()
	}()

	require.True(t, em.Emit(telemetry.MetricEvent{MessageID: "1"}))
	for i := 0; i < 50; i++ {
		em.Emit(telemetry.MetricEvent{MessageID: "x"})
	}
	assert.Greater(t, em.Drops(), int64(0))
}

func TestEmitter_RetryThenSuccess(t *testing.T) {
	sink := &recordingSink{failN: 2}
	em := telemetry.NewEmitter(sink, telemetry.Config{
		BufferSize:     10,
		BatchSize:      1,
		FlushInterval:  time.Hour,
		MaxRetries:     3,
		InitialBackoff: time.Millisecond,
	})
	em.Start(context.Background())
	defer em.Stop()

	em.Emit(telemetry.MetricEvent{MessageID: "retry-me"})
	waitForBatches(t, sink, 1)
	assert.Equal(t, int64(0), em.Drops())
}

func TestEmitter_RetryExhaustedDrops(t *testing.T) {
	sink := &recordingSink{failN: 100, failErr: errors.New("nope")}
	em := telemetry.NewEmitter(sink, telemetry.Config{
		BufferSize:     10,
		BatchSize:      2,
		FlushInterval:  time.Hour,
		MaxRetries:     2,
		InitialBackoff: time.Millisecond,
	})
	em.Start(context.Background())
	em.Emit(telemetry.MetricEvent{MessageID: "a"})
	em.Emit(telemetry.MetricEvent{MessageID: "b"})

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if em.Drops() >= 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	em.Stop()
	assert.GreaterOrEqual(t, em.Drops(), int64(2))
}

func TestEmitter_StopFlushesQueue(t *testing.T) {
	sink := &recordingSink{}
	em := telemetry.NewEmitter(sink, telemetry.Config{
		BufferSize:    10,
		BatchSize:     50,
		FlushInterval: time.Hour,
	})
	em.Start(context.Background())
	em.Emit(telemetry.MetricEvent{MessageID: "x"})
	em.Stop()

	assert.Equal(t, 1, len(sink.Batches()))
}

func TestEmitter_StartIsIdempotent(t *testing.T) {
	sink := &recordingSink{}
	em := telemetry.NewEmitter(sink, telemetry.Config{
		BatchSize:     1,
		FlushInterval: 50 * time.Millisecond,
	})
	em.Start(context.Background())
	em.Start(context.Background())
	em.Emit(telemetry.MetricEvent{MessageID: "y"})
	waitForBatches(t, sink, 1)
	em.Stop()
	em.Stop()
}

func TestHTTPSink_SendSuccess(t *testing.T) {
	gotKey := ""
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("X-Service-Key")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	sink := telemetry.NewHTTPSink(srv.URL, "k-secret", srv.Client())
	err := sink.Send(context.Background(), []telemetry.MetricEvent{
		{TenantID: "t", MessageID: "m1", Provider: "AZ", Model: "gpt", AgentType: "AGENT", Status: "SUCCESS"},
	})
	require.NoError(t, err)
	assert.Equal(t, "k-secret", gotKey)
	items, ok := gotBody["items"].([]any)
	require.True(t, ok)
	assert.Len(t, items, 1)
}

func TestHTTPSink_SendNon2xxIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"detail":"boom"}`))
	}))
	defer srv.Close()

	sink := telemetry.NewHTTPSink(srv.URL, "k", srv.Client())
	err := sink.Send(context.Background(), []telemetry.MetricEvent{{MessageID: "m"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestHTTPSink_EmptyBatchSucceeds(t *testing.T) {
	sink := telemetry.NewHTTPSink("https://nope", "k", nil)
	require.NoError(t, sink.Send(context.Background(), nil))
}

// blockingSink blocks on Send until ch is closed.
type blockingChanSink struct{ ch chan struct{} }

func (s blockingChanSink) Send(_ context.Context, _ []telemetry.MetricEvent) error {
	<-s.ch
	return nil
}
func blockingSink(ch chan struct{}) telemetry.Sink { return blockingChanSink{ch: ch} }
