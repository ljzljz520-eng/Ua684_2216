package audit

import (
	"path/filepath"
	"testing"
	"time"

	"service-request-dispatch/internal/store"
)

func TestAuditRecordAndSummary(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "audit.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	l := New(s)
	now := time.Unix(30, 0)
	if err := l.Record("u1", "create", "request", "r1", "created", now); err != nil {
		t.Fatal(err)
	}
	if err := l.Record("u1", "dispatch", "request", "r1", "assigned", now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	events, err := l.ForEntity("r1")
	if err != nil || len(events) != 2 {
		t.Fatalf("events=%v %#v", err, events)
	}
	if l.Summarize(events)["dispatch"] != 1 {
		t.Fatal("summary missing")
	}
}
