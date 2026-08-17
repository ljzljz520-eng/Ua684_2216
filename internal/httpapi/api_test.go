package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"service-request-dispatch/internal/audit"
	"service-request-dispatch/internal/model"
	"service-request-dispatch/internal/queue"
	"service-request-dispatch/internal/service"
	"service-request-dispatch/internal/store"
)

func testHandler(t *testing.T) http.Handler {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	app := service.New(s, audit.New(s), queue.New(4))
	if err := app.RegisterUser(model.UserAccount{ID: "agent", Name: "Agent", Email: "agent@example.test", Role: model.RoleAgent, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := app.CreateGroup(model.AgentGroup{ID: "g", Name: "General", Members: []string{"agent"}, Active: true}); err != nil {
		t.Fatal(err)
	}
	return New(app)
}

func TestHTTPHomeAndCreate(t *testing.T) {
	h := testHandler(t)
	page := httptest.NewRecorder()
	h.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/", nil))
	if page.Code != http.StatusOK || !bytes.Contains(page.Body.Bytes(), []byte("Service Request Dispatch")) {
		t.Fatalf("home status=%d body=%s", page.Code, page.Body.String())
	}
	payload, _ := json.Marshal(model.ServiceRequest{ID: "api-r", Subject: "API issue", Description: "api request body", Customer: "c", GroupID: "g", Priority: 3})
	record := httptest.NewRecorder()
	h.ServeHTTP(record, httptest.NewRequest(http.MethodPost, "/api/requests", bytes.NewReader(payload)))
	if record.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", record.Code, record.Body.String())
	}
}

func TestHTTPRejectsUnknownJSONAndParsesRoutes(t *testing.T) {
	h := testHandler(t)
	record := httptest.NewRecorder()
	h.ServeHTTP(record, httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewBufferString(`{"id":"u","name":"U","email":"u@example.test","role":"agent","enabled":true,"unknown":1}`)))
	if record.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d", record.Code)
	}
	route := ParseRoute("/api/requests/a%2Fb/audit")
	if route.Resource != "requests" || route.ID != "a/b" || route.Action != "audit" {
		t.Fatalf("route=%#v", route)
	}
	if !IsMutationMethod(http.MethodPatch) || IsReadMethod(http.MethodPatch) {
		t.Fatal("method classification")
	}
}
