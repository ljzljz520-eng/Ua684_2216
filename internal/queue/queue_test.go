package queue

import (
	"context"
	"testing"
)

func TestQueueEnqueueAndClose(t *testing.T) {
	q := New(2)
	if q.Capacity() != 2 {
		t.Fatalf("capacity=%d", q.Capacity())
	}
	if err := q.Enqueue(context.Background(), Job{RequestID: "r"}); err != nil {
		t.Fatal(err)
	}
	if q.Pending() != 1 {
		t.Fatalf("pending=%d", q.Pending())
	}
	q.Close()
	if err := q.Enqueue(context.Background(), Job{RequestID: "second"}); err == nil {
		t.Fatal("expected closed error")
	}
}
