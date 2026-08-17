package queue

import (
	"context"
	"sync"

	"service-request-dispatch/internal/model"
)

type Job struct {
	RequestID string
	GroupID   string
	Done      chan Result
}

type Result struct {
	RequestID string
	AgentID   string
	Err       error
}

type Queue struct {
	jobs   chan Job
	closed chan struct{}
	mu     sync.RWMutex
}

func New(size int) *Queue {
	if size < 1 {
		size = 1
	}
	return &Queue{jobs: make(chan Job, size), closed: make(chan struct{})}
}

func (q *Queue) Enqueue(ctx context.Context, job Job) error {
	if job.Done == nil {
		job.Done = make(chan Result, 1)
	}
	q.mu.RLock()
	defer q.mu.RUnlock()
	select {
	case <-q.closed:
		return model.ErrInvalidTransition
	default:
	}
	select {
	case q.jobs <- job:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (q *Queue) Start(ctx context.Context) chan struct{} {
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		for {
			select {
			case <-ctx.Done():
				return
			case <-q.closed:
				return
			case _, ok := <-q.jobs:
				if !ok {
					return
				}
			}
		}
	}()
	return finished
}

func (q *Queue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	select {
	case <-q.closed:
	default:
		close(q.closed)
	}
}

func (q *Queue) Pending() int { return len(q.jobs) }

func (q *Queue) Capacity() int { return cap(q.jobs) }
