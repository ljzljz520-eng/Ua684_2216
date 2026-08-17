package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"service-request-dispatch/internal/audit"
	"service-request-dispatch/internal/model"
	"service-request-dispatch/internal/queue"
	"service-request-dispatch/internal/store"
)

func newFixture(t *testing.T) (*Service, *store.Store) {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "service.db"))
	if err != nil {
		t.Fatal(err)
	}
	q := queue.New(8)
	app := New(s, audit.New(s), q)
	clock := time.Unix(100, 0)
	app.SetClock(func() time.Time { return clock })
	return app, s
}

func seedAgent(t *testing.T, app *Service) {
	t.Helper()
	if err := app.RegisterUser(model.UserAccount{ID: "agent-1", Name: "A One", Email: "a@example.test", Role: model.RoleAgent, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := app.CreateGroup(model.AgentGroup{ID: "group-billing", Name: "Billing", Members: []string{"agent-1"}, Active: true}); err != nil {
		t.Fatal(err)
	}
}

func validRequest(id string) model.ServiceRequest {
	return model.ServiceRequest{ID: id, Subject: "Payment issue", Description: "customer cannot pay invoice", Customer: "customer-1", GroupID: "group-billing", Priority: 4, Tags: []string{"billing"}}
}

func TestWorkflowCreateQueueDispatchResolve(t *testing.T) {
	app, s := newFixture(t)
	defer s.Close()
	seedAgent(t, app)
	request, err := app.CreateAndDispatch(context.Background(), validRequest("req-1"), "manager-1")
	if err != nil {
		t.Fatal(err)
	}
	if request.Status != model.StatusQueued {
		t.Fatalf("status=%s", request.Status)
	}
	result, err := app.ProcessQueue(context.Background(), "system")
	if err != nil || result.AgentID != "agent-1" {
		t.Fatalf("dispatch=%#v err=%v", result, err)
	}
	updated, err := app.ChangeStatus("req-1", "agent-1", model.StatusResolved)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != model.StatusResolved {
		t.Fatalf("status=%s", updated.Status)
	}
}

func TestPartialFailureReportsErrorAndCleansRecord(t *testing.T) {
	app, s := newFixture(t)
	defer s.Close()
	request := validRequest("req-partial")
	request.GroupID = "missing-group"
	_, err := app.CreateAndDispatch(context.Background(), request, "manager-1")
	if err == nil {
		t.Fatal("expected dispatch validation failure")
	}
	if _, getErr := app.GetRequest(request.ID); getErr == nil {
		t.Fatal("partial request remained after failed dispatch")
	}
}

func TestWorkflowFilterAndExport(t *testing.T) {
	app, s := newFixture(t)
	defer s.Close()
	seedAgent(t, app)
	for _, id := range []string{"req-a", "req-b"} {
		if _, err := app.CreateAndDispatch(context.Background(), validRequest(id), "manager-1"); err != nil {
			t.Fatal(err)
		}
		if _, err := app.ProcessQueue(context.Background(), "system"); err != nil {
			t.Fatal(err)
		}
	}
	items, err := app.Filter(model.RequestFilter{Status: model.StatusAssigned, GroupID: "group-billing", Tag: "BILLING"})
	if err != nil || len(items) != 2 {
		t.Fatalf("filter=%v %#v", err, items)
	}
	output, err := app.Export(model.RequestFilter{GroupID: "group-billing"})
	if err != nil || len(output) < 30 {
		t.Fatalf("export=%q err=%v", output, err)
	}
}

func TestWorkflowRejectCreatesAudit(t *testing.T) {
	app, s := newFixture(t)
	defer s.Close()
	seedAgent(t, app)
	if _, err := app.CreateAndDispatch(context.Background(), validRequest("req-reject"), "manager-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.ProcessQueue(context.Background(), "system"); err != nil {
		t.Fatal(err)
	}
	if _, err := app.ChangeStatus("req-reject", "manager-1", model.StatusRejected); err != nil {
		t.Fatal(err)
	}
	events, err := app.Audits("req-reject")
	if err != nil || len(events) < 2 {
		t.Fatalf("audit=%v %#v", err, events)
	}
}
