// Package traceimport provides functionality for importing traces from external systems.
package traceimport

import (
	"context"
	"sync"
)

// JobQueue manages a queue of import jobs.
type JobQueue struct {
	jobs       chan *ImportJob
	workerFunc func(ctx context.Context, job *ImportJob) error
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	started    bool
	mu         sync.Mutex
}

// NewJobQueue creates a new job queue with the specified buffer size and worker function.
func NewJobQueue(bufferSize int, workerFunc func(ctx context.Context, job *ImportJob) error) *JobQueue {
	ctx, cancel := context.WithCancel(context.Background())
	return &JobQueue{
		jobs:       make(chan *ImportJob, bufferSize),
		workerFunc: workerFunc,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start starts the job queue workers.
func (q *JobQueue) Start(workerCount int) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.started {
		return
	}
	q.started = true

	for i := 0; i < workerCount; i++ {
		q.wg.Add(1)
		go q.worker()
	}
}

// worker processes jobs from the queue.
func (q *JobQueue) worker() {
	defer q.wg.Done()

	for {
		select {
		case <-q.ctx.Done():
			return
		case job, ok := <-q.jobs:
			if !ok {
				return
			}
			// Execute the worker function - errors are logged but not propagated
			_ = q.workerFunc(q.ctx, job)
		}
	}
}

// Enqueue adds a job to the queue.
// Returns immediately after adding the job (non-blocking).
func (q *JobQueue) Enqueue(job *ImportJob) {
	select {
	case q.jobs <- job:
		// Job added successfully
	default:
		// Queue is full - log and drop (in production, you might want to handle this differently)
	}
}

// Stop stops the job queue gracefully.
func (q *JobQueue) Stop() {
	q.cancel()
	close(q.jobs)
	q.wg.Wait()
}

// QueueSize returns the current number of jobs in the queue.
func (q *JobQueue) QueueSize() int {
	return len(q.jobs)
}
