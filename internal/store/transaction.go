package store

import (
	"fmt"
	"time"

	"go.etcd.io/bbolt"
	"service-request-dispatch/internal/model"
)

type RequestDraft struct {
	Request model.ServiceRequest
	Attempt model.DispatchAttempt
}

func (s *Store) SaveDispatchDraft(draft RequestDraft) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.db.Update(func(tx *bbolt.Tx) error {
		if err := put(tx, []byte("service_requests"), draft.Request.ID, draft.Request); err != nil {
			return err
		}
		if draft.Attempt.ID == "" {
			return fmt.Errorf("dispatch attempt id is required")
		}
		if err := put(tx, []byte("dispatch_attempts"), draft.Attempt.ID, draft.Attempt); err != nil {
			return err
		}
		return nil
	})
}

func (s *Store) UpdateRequestStatus(id string, status model.RequestStatus, groupID, assignee string, now time.Time) (model.ServiceRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var updated model.ServiceRequest
	err := s.db.Update(func(tx *bbolt.Tx) error {
		if err := get(tx, []byte("service_requests"), id, &updated); err != nil {
			return err
		}
		if !model.CanTransition(updated.Status, status) {
			return model.ErrInvalidTransition
		}
		updated.Status = status
		updated.GroupID = groupID
		updated.Assignee = assignee
		updated.UpdatedAt = now
		return put(tx, []byte("service_requests"), id, updated)
	})
	return updated, err
}
