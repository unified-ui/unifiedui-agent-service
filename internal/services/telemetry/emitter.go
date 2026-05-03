// Package telemetry provides an asynchronous emitter that buffers per-message
// metric events and forwards them to the platform-service internal ingest API
// (POST /api/v1/platform-service/internal/metrics/messages:batch).
//
// The emitter is non-blocking: when the buffer is full, events are dropped and
// counted via TelemetryDropsTotal. This guarantees that telemetry never adds
// latency to the request-serving path.
package telemetry

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// MetricEvent is a single per-message telemetry record submitted by callers.
type MetricEvent struct {
	TenantID       string `json:"tenant_id"`
	MessageID      string `json:"message_id"`
	ChatAgentID    string `json:"chat_agent_id,omitempty"`
	WorkflowID     string `json:"workflow_id,omitempty"`
	ConversationID string `json:"conversation_id,omitempty"`
	UserID         string `json:"user_id,omitempty"`
	Provider       string `json:"provider"`
	Model          string `json:"model"`
	TokensInput    int    `json:"tokens_input"`
	TokensOutput   int    `json:"tokens_output"`
	LatencyMs      int    `json:"latency_ms"`
	AgentType      string `json:"agent_type"`
	Status         string `json:"status"`
	ErrorCode      string `json:"error_code,omitempty"`
}

// Sink delivers a batch of MetricEvents to the platform-service.
//
// Implementations must be safe for concurrent use. A non-nil error indicates
// the batch should be retried (with exponential back-off, capped at the
// configured retry count).
type Sink interface {
	Send(ctx context.Context, batch []MetricEvent) error
}

// Config controls the emitter's batching, retry, and back-pressure behaviour.
type Config struct {
	// BufferSize bounds the in-memory queue. Defaults to 1000.
	BufferSize int
	// FlushInterval is the maximum time a batch can sit in the buffer before
	// being flushed even if BatchSize is not reached. Defaults to 5s.
	FlushInterval time.Duration
	// BatchSize is the size at which a partial batch is flushed early.
	// Defaults to 100 (matches the platform-service batch limit).
	BatchSize int
	// MaxRetries caps the number of retry attempts for a failed batch.
	// Defaults to 3.
	MaxRetries int
	// InitialBackoff is the first sleep before retry #1.
	// Defaults to 200ms (subsequent attempts double the delay).
	InitialBackoff time.Duration
}

func (c *Config) withDefaults() Config {
	out := *c
	if out.BufferSize <= 0 {
		out.BufferSize = 1000
	}
	if out.FlushInterval <= 0 {
		out.FlushInterval = 5 * time.Second
	}
	if out.BatchSize <= 0 {
		out.BatchSize = 100
	}
	if out.MaxRetries < 0 {
		out.MaxRetries = 0
	}
	if out.MaxRetries == 0 {
		out.MaxRetries = 3
	}
	if out.InitialBackoff <= 0 {
		out.InitialBackoff = 200 * time.Millisecond
	}
	return out
}

// Emitter buffers MetricEvents and forwards them to the configured Sink.
//
// The zero value is not usable; create one via NewEmitter.
type Emitter struct {
	cfg     Config
	sink    Sink
	queue   chan MetricEvent
	stopCh  chan struct{}
	doneCh  chan struct{}
	started atomic.Bool
	stopped atomic.Bool

	dropsMu sync.Mutex
	drops   int64
	sleepFn func(d time.Duration)
}

// NewEmitter constructs a new Emitter. Sink must be non-nil.
func NewEmitter(sink Sink, cfg Config) *Emitter {
	c := cfg.withDefaults()
	return &Emitter{
		cfg:     c,
		sink:    sink,
		queue:   make(chan MetricEvent, c.BufferSize),
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
		sleepFn: time.Sleep,
	}
}

// Start launches the background worker. Subsequent calls are no-ops.
func (e *Emitter) Start(ctx context.Context) {
	if !e.started.CompareAndSwap(false, true) {
		return
	}
	go e.run(ctx)
}

// Stop signals the worker to flush its current batch and exit. Safe to call
// multiple times. Blocks until the worker has finished.
func (e *Emitter) Stop() {
	if !e.stopped.CompareAndSwap(false, true) {
		return
	}
	close(e.stopCh)
	<-e.doneCh
}

// Emit enqueues a single MetricEvent. If the buffer is full, the event is
// dropped (counted via Drops()). Emit never blocks.
func (e *Emitter) Emit(event MetricEvent) bool {
	select {
	case e.queue <- event:
		return true
	default:
		e.dropsMu.Lock()
		e.drops++
		e.dropsMu.Unlock()
		return false
	}
}

// Drops returns the cumulative number of events that were dropped due to
// back-pressure.
func (e *Emitter) Drops() int64 {
	e.dropsMu.Lock()
	defer e.dropsMu.Unlock()
	return e.drops
}

// run is the background worker loop. It flushes whenever BatchSize is reached,
// FlushInterval elapses, or Stop is invoked.
func (e *Emitter) run(ctx context.Context) {
	defer close(e.doneCh)

	ticker := time.NewTicker(e.cfg.FlushInterval)
	defer ticker.Stop()

	batch := make([]MetricEvent, 0, e.cfg.BatchSize)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		e.sendWithRetry(ctx, batch)
		batch = batch[:0]
	}

	for {
		select {
		case <-e.stopCh:
			e.drainAndFlush(ctx, batch)
			return
		case <-ctx.Done():
			e.drainAndFlush(ctx, batch)
			return
		case ev := <-e.queue:
			batch = append(batch, ev)
			if len(batch) >= e.cfg.BatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (e *Emitter) drainAndFlush(ctx context.Context, batch []MetricEvent) {
	for {
		select {
		case ev := <-e.queue:
			batch = append(batch, ev)
			if len(batch) >= e.cfg.BatchSize {
				e.sendWithRetry(ctx, batch)
				batch = batch[:0]
			}
		default:
			if len(batch) > 0 {
				e.sendWithRetry(ctx, batch)
			}
			return
		}
	}
}

func (e *Emitter) sendWithRetry(ctx context.Context, batch []MetricEvent) {
	dispatch := make([]MetricEvent, len(batch))
	copy(dispatch, batch)

	delay := e.cfg.InitialBackoff
	for attempt := 0; attempt <= e.cfg.MaxRetries; attempt++ {
		err := e.sink.Send(ctx, dispatch)
		if err == nil {
			return
		}
		if attempt == e.cfg.MaxRetries {
			e.dropsMu.Lock()
			e.drops += int64(len(dispatch))
			e.dropsMu.Unlock()
			return
		}
		e.sleepFn(delay)
		delay *= 2
	}
}
