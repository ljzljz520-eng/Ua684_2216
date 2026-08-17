package store

import (
	"encoding/json"
	"fmt"
	"time"

	"go.etcd.io/bbolt"
	"service-request-dispatch/internal/model"
)

type Snapshot struct {
	Requests   []model.ServiceRequest  `json:"requests"`
	Groups     []model.AgentGroup      `json:"groups"`
	Users      []model.UserAccount     `json:"users"`
	Attempts   []model.DispatchAttempt `json:"attempts"`
	Audits     []model.AuditEvent      `json:"audits"`
	CapturedAt time.Time               `json:"captured_at"`
}

func (s *Store) Snapshot() (Snapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := Snapshot{Requests: make([]model.ServiceRequest, 0), Groups: make([]model.AgentGroup, 0), Users: make([]model.UserAccount, 0), Attempts: make([]model.DispatchAttempt, 0), Audits: make([]model.AuditEvent, 0), CapturedAt: time.Unix(0, 0)}
	err := s.db.View(func(tx *bbolt.Tx) error {
		if err := collect(tx.Bucket([]byte("service_requests")), func(data []byte) error {
			var v model.ServiceRequest
			if err := decode(data, &v); err != nil {
				return err
			}
			result.Requests = append(result.Requests, v)
			return nil
		}); err != nil {
			return err
		}
		if err := collect(tx.Bucket([]byte("agent_groups")), func(data []byte) error {
			var v model.AgentGroup
			if err := decode(data, &v); err != nil {
				return err
			}
			result.Groups = append(result.Groups, v)
			return nil
		}); err != nil {
			return err
		}
		if err := collect(tx.Bucket([]byte("user_accounts")), func(data []byte) error {
			var v model.UserAccount
			if err := decode(data, &v); err != nil {
				return err
			}
			result.Users = append(result.Users, v)
			return nil
		}); err != nil {
			return err
		}
		if err := collect(tx.Bucket([]byte("dispatch_attempts")), func(data []byte) error {
			var v model.DispatchAttempt
			if err := decode(data, &v); err != nil {
				return err
			}
			result.Attempts = append(result.Attempts, v)
			return nil
		}); err != nil {
			return err
		}
		return collect(tx.Bucket([]byte("audit_events")), func(data []byte) error {
			var v model.AuditEvent
			if err := decode(data, &v); err != nil {
				return err
			}
			result.Audits = append(result.Audits, v)
			return nil
		})
	})
	return result, err
}

func collect(bucket *bbolt.Bucket, consume func([]byte) error) error {
	if bucket == nil {
		return fmt.Errorf("snapshot bucket missing")
	}
	return bucket.ForEach(func(_, value []byte) error { return consume(value) })
}

func (s *Store) ExportSnapshot() ([]byte, error) {
	snapshot, err := s.Snapshot()
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(snapshot, "", "  ")
}

func (s *Store) ImportSnapshot(data []byte) error {
	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return fmt.Errorf("decode snapshot: %w", err)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.db.Update(func(tx *bbolt.Tx) error {
		for _, request := range snapshot.Requests {
			if err := put(tx, []byte("service_requests"), request.ID, request); err != nil {
				return err
			}
		}
		for _, group := range snapshot.Groups {
			if err := put(tx, []byte("agent_groups"), group.ID, group); err != nil {
				return err
			}
		}
		for _, user := range snapshot.Users {
			if err := put(tx, []byte("user_accounts"), user.ID, user); err != nil {
				return err
			}
		}
		for _, attempt := range snapshot.Attempts {
			if err := put(tx, []byte("dispatch_attempts"), attempt.ID, attempt); err != nil {
				return err
			}
		}
		for _, event := range snapshot.Audits {
			if err := put(tx, []byte("audit_events"), event.ID, event); err != nil {
				return err
			}
		}
		return nil
	})
}
