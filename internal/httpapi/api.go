package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"service-request-dispatch/internal/model"
	"service-request-dispatch/internal/service"
)

type API struct {
	app  *service.Service
	mux  *http.ServeMux
	page template.Template
}

func New(app *service.Service) http.Handler {
	api := &API{app: app, mux: http.NewServeMux(), page: *template.Must(template.New("index").Parse(indexPage))}
	api.routes()
	return logging(api.mux)
}

func (a *API) routes() {
	a.mux.HandleFunc("/", a.home)
	a.mux.HandleFunc("/health", a.health)
	a.mux.HandleFunc("/api/users", a.users)
	a.mux.HandleFunc("/api/groups", a.groups)
	a.mux.HandleFunc("/api/requests", a.requests)
	a.mux.HandleFunc("/api/requests/", a.requestAction)
	a.mux.HandleFunc("/api/export", a.export)
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Service", "request-dispatch")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func (a *API) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	items, err := a.app.Filter(model.RequestFilter{})
	if err != nil {
		writeError(w, err)
		return
	}
	data := struct {
		Requests  []model.ServiceRequest
		Dashboard service.Dashboard
	}{Requests: items, Dashboard: service.BuildDashboard(items)}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.page.Execute(w, data); err != nil {
		return
	}
}

func (a *API) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	if err := a.app.Health(); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *API) users(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var user model.UserAccount
	if err := readJSON(r, &user); err != nil {
		writeError(w, err)
		return
	}
	if err := user.Validate(); err != nil {
		writeBusinessError(w, err)
		return
	}
	if err := a.app.RegisterUser(user); err != nil {
		writeBusinessError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

func (a *API) groups(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		groups, err := a.app.ListGroups()
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, groups)
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var group model.AgentGroup
	if err := readJSON(r, &group); err != nil {
		writeError(w, err)
		return
	}
	if err := a.app.CreateGroup(group); err != nil {
		writeBusinessError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, group)
}

func (a *API) requests(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		query := parseFilter(r)
		items, err := a.app.Filter(query)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"requests": items, "dashboard": service.BuildDashboard(items)})
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var request model.ServiceRequest
	if err := readJSON(r, &request); err != nil {
		writeError(w, err)
		return
	}
	actor := r.Header.Get("X-Actor")
	request, err := a.app.CreateAndDispatch(r.Context(), request, actor)
	if err != nil {
		writeBusinessError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, request)
}

func (a *API) requestAction(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/requests/"), "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	id := parts[0]
	if len(parts) == 1 && r.Method == http.MethodGet {
		request, err := a.app.GetRequest(id)
		if err != nil {
			writeBusinessError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, request)
		return
	}
	if len(parts) == 2 && parts[1] == "process" && r.Method == http.MethodPost {
		result, err := a.app.ProcessQueue(r.Context(), r.Header.Get("X-Actor"))
		if err != nil {
			writeBusinessError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
		return
	}
	if len(parts) == 2 && parts[1] == "status" && r.Method == http.MethodPatch {
		a.changeStatus(w, r, id)
		return
	}
	if len(parts) == 2 && parts[1] == "audit" && r.Method == http.MethodGet {
		events, err := a.app.Audits(id)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, events)
		return
	}
	if len(parts) == 2 && parts[1] == "attempts" && r.Method == http.MethodGet {
		attempts, err := a.app.Attempts(id)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, attempts)
		return
	}
	http.NotFound(w, r)
}

func (a *API) changeStatus(w http.ResponseWriter, r *http.Request, id string) {
	var input struct {
		Status string `json:"status"`
	}
	if err := readJSON(r, &input); err != nil {
		writeError(w, err)
		return
	}
	status := model.NormalizeStatus(input.Status)
	if status == model.StatusQueued && input.Status != string(model.StatusQueued) {
		writeBusinessError(w, model.ErrInvalidTransition)
		return
	}
	updated, err := a.app.ChangeStatus(id, r.Header.Get("X-Actor"), status)
	if err != nil {
		writeBusinessError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (a *API) export(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	data, err := a.app.Export(parseFilter(r))
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=service-requests.csv")
	_, _ = w.Write([]byte(data))
}

func parseFilter(r *http.Request) model.RequestFilter {
	query := r.URL.Query()
	result := model.RequestFilter{Status: model.RequestStatus(query.Get("status")), GroupID: query.Get("group"), Assignee: query.Get("assignee"), Tag: query.Get("tag")}
	if value, err := strconv.Atoi(query.Get("priority")); err == nil && value > 0 {
		result.Priority = value
	}
	if value, err := time.Parse(time.RFC3339, query.Get("from")); err == nil {
		result.From = value
	}
	if value, err := time.Parse(time.RFC3339, query.Get("to")); err == nil {
		result.To = value
	}
	return result
}

func readJSON(r *http.Request, target any) error {
	if r.Body == nil {
		return errors.New("request body is required")
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
}

func writeBusinessError(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error(), "kind": "business"})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
}

const indexPage = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>Service Request Dispatch</title>
<style>body{font:16px system-ui;margin:2rem;background:#f6f7f9;color:#17202a}main{max-width:1100px;margin:auto}header{display:flex;justify-content:space-between;align-items:center;border-bottom:1px solid #ccd3db;margin-bottom:1rem}table{width:100%;border-collapse:collapse;background:white}th,td{padding:.6rem;border-bottom:1px solid #e1e5e9;text-align:left}aside{display:flex;gap:1rem;margin:1rem 0}strong{font-size:1.5rem}</style></head>
<body><main><header><h1>Service Request Dispatch</h1><span>local operations console</span></header><aside><div><strong>{{.Dashboard.Total}}</strong><br>total</div><div><strong>{{.Dashboard.Open}}</strong><br>open</div></aside><table><thead><tr><th>Subject</th><th>Customer</th><th>Status</th><th>Group</th><th>Priority</th><th>Created</th></tr></thead><tbody>{{range .Requests}}<tr><td>{{.Subject}}</td><td>{{.Customer}}</td><td>{{.Status}}</td><td>{{.GroupID}}</td><td>{{.Priority}}</td><td>{{.CreatedAt}}</td></tr>{{else}}<tr><td colspan="6">No service requests</td></tr>{{end}}</tbody></table></main></body></html>`

func _contextUsed(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
