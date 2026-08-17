package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"service-request-dispatch/internal/audit"
	"service-request-dispatch/internal/filter"
	"service-request-dispatch/internal/model"
	"service-request-dispatch/internal/queue"
	"service-request-dispatch/internal/store"
)

type Service struct {
	store *store.Store
	audit *audit.Logger
	queue *queue.Queue
	now   func() time.Time
	mu    sync.Mutex
}

func New(s *store.Store, logger *audit.Logger, q *queue.Queue) *Service {
	return &Service{store: s, audit: logger, queue: q, now: time.Now}
}

func (s *Service) SetClock(now func() time.Time) {
	if now != nil {
		s.now = now
	}
}

func (s *Service) RegisterUser(user model.UserAccount) error {
	if user.ID == "" || user.Name == "" || user.Email == "" {
		return fmt.Errorf("user identity is incomplete")
	}
	if user.Role != model.RoleAgent && user.Role != model.RoleManager && user.Role != model.RoleAuditor && user.Role != model.RoleAdmin {
		return fmt.Errorf("unsupported user role")
	}
	if err := s.store.PutUser(user); err != nil {
		return err
	}
	return s.audit.Record(user.ID, "user.registered", "user", user.ID, string(user.Role), s.now())
}

func (s *Service) GetUser(id string) (model.UserAccount, error) { return s.store.GetUser(id) }

func (s *Service) CreateGroup(group model.AgentGroup) error {
	if group.ID == "" || strings.TrimSpace(group.Name) == "" {
		return fmt.Errorf("group identity is incomplete")
	}
	if !group.Active {
		return model.ErrGroupInactive
	}
	if len(group.Members) == 0 {
		return model.ErrNoAvailableAgent
	}
	for _, member := range group.Members {
		user, err := s.store.GetUser(member)
		if err != nil {
			return fmt.Errorf("group member %s: %w", member, err)
		}
		if user.Role != model.RoleAgent || !user.Enabled {
			return fmt.Errorf("group member %s is not an enabled agent", member)
		}
	}
	if err := s.store.PutGroup(group); err != nil {
		return err
	}
	return s.audit.Record("system", "group.created", "group", group.ID, group.Name, s.now())
}

func (s *Service) GetGroup(id string) (model.AgentGroup, error) { return s.store.GetGroup(id) }

func (s *Service) ListGroups() ([]model.AgentGroup, error) { return s.store.ListGroups() }

func (s *Service) CreateAndDispatch(ctx context.Context, request model.ServiceRequest, actor string) (model.ServiceRequest, error) {
	if request.Status == "" {
		request.Status = model.StatusQueued
	}
	if request.CreatedAt.IsZero() {
		request.CreatedAt = s.now()
	}
	request.UpdatedAt = request.CreatedAt
	if err := request.Validate(); err != nil {
		return request, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.store.GetRequest(request.ID); err == nil {
		return request, model.ErrAlreadyExists
	}
	if err := s.store.PutRequest(request); err != nil {
		return request, err
	}
	if err := s.validateDispatchTarget(request.GroupID); err != nil {
		// Step 1 already persisted the request; step 2 (business validation)
		// rejected it. Roll back the half-finished record and surface the
		// recognizable business failure so callers can act on the cause.
		_ = s.store.DeleteRequest(request.ID)
		return request, err
	}
	job := queue.Job{RequestID: request.ID, GroupID: request.GroupID, Done: make(chan queue.Result, 1)}
	if err := s.queue.Enqueue(ctx, job); err != nil {
		_ = s.store.DeleteRequest(request.ID)
		return request, err
	}
	if err := s.audit.Record(actor, "request.created", "request", request.ID, "queued", s.now()); err != nil {
		return request, err
	}
	return request, nil
}

func (s *Service) validateDispatchTarget(groupID string) error {
	if groupID == "" {
		return model.ErrGroupNotFound
	}
	group, err := s.store.GetGroup(groupID)
	if err != nil {
		return err
	}
	if !group.Active {
		return model.ErrGroupInactive
	}
	if len(group.Members) == 0 {
		return model.ErrNoAvailableAgent
	}
	for _, member := range group.Members {
		user, userErr := s.store.GetUser(member)
		if userErr == nil && user.Enabled && user.Role == model.RoleAgent {
			return nil
		}
	}
	return model.ErrNoAvailableAgent
}

func (s *Service) ProcessQueue(ctx context.Context, actor string) (queue.Result, error) {
	job, ok := s.queue.Receive(ctx)
	if !ok {
		return queue.Result{}, context.Canceled
	}
	request, err := s.store.GetRequest(job.RequestID)
	if err != nil {
		return queue.Result{RequestID: job.RequestID, Err: err}, err
	}
	agent, err := s.chooseAgent(job.GroupID)
	if err != nil {
		_ = s.audit.Record(actor, "request.dispatch_failed", "request", request.ID, err.Error(), s.now())
		return queue.Result{RequestID: request.ID, Err: err}, err
	}
	if _, err := s.store.UpdateRequestStatus(request.ID, model.StatusAssigned, job.GroupID, agent, s.now()); err != nil {
		return queue.Result{RequestID: request.ID, Err: err}, err
	}
	attempt := model.DispatchAttempt{ID: fmt.Sprintf("attempt-%s-%d", request.ID, s.now().UnixNano()), RequestID: request.ID, GroupID: job.GroupID, AgentID: agent, Outcome: "assigned", CreatedAt: s.now()}
	if err := s.store.PutAttempt(attempt); err != nil {
		return queue.Result{RequestID: request.ID, AgentID: agent, Err: err}, err
	}
	if err := s.audit.Record(actor, "request.dispatched", "request", request.ID, agent, s.now()); err != nil {
		return queue.Result{RequestID: request.ID, AgentID: agent, Err: err}, err
	}
	return queue.Result{RequestID: request.ID, AgentID: agent}, nil
}

func (s *Service) chooseAgent(groupID string) (string, error) {
	group, err := s.store.GetGroup(groupID)
	if err != nil {
		return "", err
	}
	if !group.Active {
		return "", model.ErrGroupInactive
	}
	for _, member := range group.Members {
		user, userErr := s.store.GetUser(member)
		if userErr == nil && user.Enabled && user.Role == model.RoleAgent {
			return member, nil
		}
	}
	return "", model.ErrNoAvailableAgent
}

func (s *Service) ChangeStatus(id, actor string, next model.RequestStatus) (model.ServiceRequest, error) {
	request, err := s.store.GetRequest(id)
	if err != nil {
		return request, err
	}
	if request.IsTerminal() {
		return request, model.ErrInvalidTransition
	}
	if next == model.StatusResolved && actor == "" {
		return request, model.ErrPermissionDenied
	}
	updated, err := s.store.UpdateRequestStatus(id, next, request.GroupID, request.Assignee, s.now())
	if err != nil {
		return request, err
	}
	if err := s.audit.Record(actor, "request.status_changed", "request", id, string(next), s.now()); err != nil {
		return updated, err
	}
	return updated, nil
}

func (s *Service) Filter(query model.RequestFilter) ([]model.ServiceRequest, error) {
	items, err := s.store.ListRequests()
	if err != nil {
		return nil, err
	}
	return filter.Apply(items, query), nil
}

func (s *Service) GetRequest(id string) (model.ServiceRequest, error) { return s.store.GetRequest(id) }

func (s *Service) Attempts(id string) ([]model.DispatchAttempt, error) {
	return s.store.ListAttempts(id)
}

func (s *Service) Audits(id string) ([]model.AuditEvent, error) { return s.audit.ForEntity(id) }

func (s *Service) RequireRole(userID string, roles ...model.UserRole) error {
	user, err := s.store.GetUser(userID)
	if err != nil {
		return err
	}
	for _, role := range roles {
		if user.Role == role && user.Enabled {
			return nil
		}
	}
	return model.ErrPermissionDenied
}

func (s *Service) Export(query model.RequestFilter) (string, error) {
	items, err := s.Filter(query)
	if err != nil {
		return "", err
	}
	var builder strings.Builder
	builder.WriteString("id,subject,customer,status,group,assignee,priority,created_at\n")
	for _, item := range items {
		builder.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%d,%s\n", csv(item.ID), csv(item.Subject), csv(item.Customer), item.Status, csv(item.GroupID), csv(item.Assignee), item.Priority, item.CreatedAt.UTC().Format(time.RFC3339)))
	}
	return builder.String(), nil
}

func csv(value string) string {
	if strings.ContainsAny(value, ",\"\n") {
		return "\"" + strings.ReplaceAll(value, "\"", "\"\"") + "\""
	}
	return value
}

func (s *Service) Health() error {
	_, err := s.store.Count("service_requests")
	return err
}

func IsBusinessError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, model.ErrGroupNotFound) || errors.Is(err, model.ErrGroupInactive) || errors.Is(err, model.ErrNoAvailableAgent) || errors.Is(err, model.ErrInvalidTransition)
}
