package audit

import (
	"fmt"
	"time"

	"service-request-dispatch/internal/model"
	"service-request-dispatch/internal/store"
)

type Logger struct{ store *store.Store }

func New(s *store.Store) *Logger { return &Logger{store: s} }

func (l *Logger) Record(actor, action, entityType, entityID, details string, now time.Time) error {
	if actor == "" {
		actor = "system"
	}
	event := model.AuditEvent{ID: fmt.Sprintf("audit-%d-%s-%s", now.UnixNano(), entityID, action), ActorID: actor, Action: action, EntityType: entityType, EntityID: entityID, Details: details, CreatedAt: now}
	return l.store.PutAudit(event)
}

func (l *Logger) ForEntity(entityID string) ([]model.AuditEvent, error) {
	return l.store.ListAudits(entityID)
}

func (l *Logger) Summarize(events []model.AuditEvent) map[string]int {
	result := make(map[string]int)
	for _, event := range events {
		result[event.Action]++
	}
	return result
}

func (l *Logger) ValidateAction(action string) error {
	if action == "" {
		return fmt.Errorf("audit action is required")
	}
	if len(action) > 80 {
		return fmt.Errorf("audit action is too long")
	}
	return nil
}
