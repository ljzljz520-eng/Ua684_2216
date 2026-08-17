package store

import (
	"path/filepath"
	"testing"
	"time"

	"service-request-dispatch/internal/model"
)

func TestPersistenceSurvivesReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "requests.db")
	first, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	request := model.ServiceRequest{ID: "req-reopen", Subject: "reopen", Description: "persisted request", Customer: "cust", Priority: 3, Status: model.StatusQueued, CreatedAt: time.Unix(10, 0), UpdatedAt: time.Unix(10, 0)}
	if err := first.PutRequest(request); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	got, err := second.GetRequest(request.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Subject != request.Subject || got.Status != model.StatusQueued {
		t.Fatalf("unexpected reopened request: %#v", got)
	}
}

func TestStoreListsRequestsInCreationOrder(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "list.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, item := range []model.ServiceRequest{{ID: "b", Subject: "b", Description: "second item", Customer: "c", Priority: 2, CreatedAt: time.Unix(20, 0)}, {ID: "a", Subject: "a", Description: "first item", Customer: "c", Priority: 1, CreatedAt: time.Unix(10, 0)}} {
		if err := s.PutRequest(item); err != nil {
			t.Fatal(err)
		}
	}
	items, err := s.ListRequests()
	if err != nil || len(items) != 2 || items[0].ID != "a" {
		t.Fatalf("list failed: %v %#v", err, items)
	}
}
