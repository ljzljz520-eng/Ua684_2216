package queue

import "context"

func (q *Queue) Receive(ctx context.Context) (Job, bool) {
	select {
	case <-ctx.Done():
		return Job{}, false
	case <-q.closed:
		return Job{}, false
	case job, ok := <-q.jobs:
		return job, ok
	}
}
