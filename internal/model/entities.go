package model

import "time"

type RequestStatus string

const (
	StatusQueued   RequestStatus = "queued"
	StatusAssigned RequestStatus = "assigned"
	StatusPending  RequestStatus = "pending"
	StatusResolved RequestStatus = "resolved"
	StatusRejected RequestStatus = "rejected"
)

type UserRole string

const (
	RoleAgent   UserRole = "agent"
	RoleManager UserRole = "manager"
	RoleAuditor UserRole = "auditor"
	RoleAdmin   UserRole = "admin"
)

type ServiceRequest struct {
	ID          string        `json:"id"`
	Subject     string        `json:"subject"`
	Description string        `json:"description"`
	Customer    string        `json:"customer"`
	Status      RequestStatus `json:"status"`
	GroupID     string        `json:"group_id"`
	Assignee    string        `json:"assignee"`
	Priority    int           `json:"priority"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	Tags        []string      `json:"tags"`
}

type AgentGroup struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Members     []string `json:"members"`
	Active      bool     `json:"active"`
}

type UserAccount struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Email    string   `json:"email"`
	Role     UserRole `json:"role"`
	GroupIDs []string `json:"group_ids"`
	Enabled  bool     `json:"enabled"`
}

type AuditEvent struct {
	ID         string    `json:"id"`
	ActorID    string    `json:"actor_id"`
	Action     string    `json:"action"`
	EntityType string    `json:"entity_type"`
	EntityID   string    `json:"entity_id"`
	Details    string    `json:"details"`
	CreatedAt  time.Time `json:"created_at"`
}

type DispatchAttempt struct {
	ID        string    `json:"id"`
	RequestID string    `json:"request_id"`
	GroupID   string    `json:"group_id"`
	AgentID   string    `json:"agent_id"`
	Outcome   string    `json:"outcome"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

type RequestFilter struct {
	Status   RequestStatus
	GroupID  string
	From     time.Time
	To       time.Time
	Assignee string
	Priority int
	Tag      string
}

func (r ServiceRequest) IsOpen() bool {
	return r.Status == StatusQueued || r.Status == StatusAssigned || r.Status == StatusPending
}

func (r ServiceRequest) IsTerminal() bool {
	return r.Status == StatusResolved || r.Status == StatusRejected
}

func (r ServiceRequest) Validate() error {
	if r.ID == "" {
		return ErrMissingRequestID
	}
	if r.Subject == "" {
		return ErrMissingSubject
	}
	if r.Customer == "" {
		return ErrMissingCustomer
	}
	if len(r.Description) < 5 {
		return ErrDescriptionTooShort
	}
	if r.Priority < 1 || r.Priority > 5 {
		return ErrInvalidPriority
	}
	return nil
}
