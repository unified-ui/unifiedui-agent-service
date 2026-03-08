package traceimport_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/services/traceimport"
)

func TestJobQueue_Enqueue_And_Process(t *testing.T) {
	var processed atomic.Int32
	queue := traceimport.NewJobQueue(10, func(ctx context.Context, job *traceimport.ImportJob) error {
		processed.Add(1)
		return nil
	})

	queue.Start(1)
	defer queue.Stop()

	queue.Enqueue(&traceimport.ImportJob{Type: traceimport.JobTypeN8N})
	queue.Enqueue(&traceimport.ImportJob{Type: traceimport.JobTypeMicrosoftFoundry})

	require.Eventually(t, func() bool {
		return processed.Load() == 2
	}, 2*time.Second, 10*time.Millisecond)
}

func TestJobQueue_QueueSize(t *testing.T) {
	queue := traceimport.NewJobQueue(10, func(ctx context.Context, job *traceimport.ImportJob) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	queue.Enqueue(&traceimport.ImportJob{})
	require.Equal(t, 1, queue.QueueSize())
}

func TestJobQueue_StartIdempotent(t *testing.T) {
	var processed atomic.Int32
	queue := traceimport.NewJobQueue(10, func(ctx context.Context, job *traceimport.ImportJob) error {
		processed.Add(1)
		return nil
	})

	queue.Start(2)
	queue.Start(2)

	queue.Enqueue(&traceimport.ImportJob{})

	require.Eventually(t, func() bool {
		return processed.Load() >= 1
	}, 2*time.Second, 10*time.Millisecond)

	queue.Stop()
}

func TestJobQueue_StopDrains(t *testing.T) {
	var processed atomic.Int32
	queue := traceimport.NewJobQueue(10, func(ctx context.Context, job *traceimport.ImportJob) error {
		processed.Add(1)
		return nil
	})

	queue.Start(1)
	queue.Enqueue(&traceimport.ImportJob{})
	time.Sleep(50 * time.Millisecond)
	queue.Stop()

	require.GreaterOrEqual(t, processed.Load(), int32(1))
}

func TestJobQueue_FullQueueDrops(t *testing.T) {
	queue := traceimport.NewJobQueue(1, func(ctx context.Context, job *traceimport.ImportJob) error {
		time.Sleep(time.Second)
		return nil
	})

	queue.Enqueue(&traceimport.ImportJob{})
	queue.Enqueue(&traceimport.ImportJob{})
	require.Equal(t, 1, queue.QueueSize())
}
