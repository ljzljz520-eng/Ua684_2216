package store

import (
	"path/filepath"
	"testing"

	"service-request-dispatch/internal/model"
)

func TestSnapshotRoundTrip(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "snapshot.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.PutRequest(model.ServiceRequest{ID: "snapshot-r", Subject: "snapshot", Description: "snapshot value", Customer: "c", Priority: 2}); err != nil {
		t.Fatal(err)
	}
	data, err := s.ExportSnapshot()
	if err != nil || len(data) == 0 {
		t.Fatalf("export=%v bytes=%d", err, len(data))
	}
	other, err := Open(filepath.Join(t.TempDir(), "import.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	if err := other.ImportSnapshot(data); err != nil {
		t.Fatal(err)
	}
	got, err := other.GetRequest("snapshot-r")
	if err != nil || got.Subject != "snapshot" {
		t.Fatalf("import=%#v err=%v", got, err)
	}
}
