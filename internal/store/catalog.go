package store

import (
	"errors"
	"sort"
	"time"

	"go.etcd.io/bbolt"
	"service-request-dispatch/internal/model"
)

func (s *Store) ListUsers() ([]model.UserAccount, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	users := make([]model.UserAccount, 0)
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("user_accounts"))
		if bucket == nil {
			return errors.New("user bucket not found")
		}
		return bucket.ForEach(func(_, data []byte) error {
			var user model.UserAccount
			if err := decode(data, &user); err != nil {
				return err
			}
			users = append(users, user)
			return nil
		})
	})
	sort.Slice(users, func(i, j int) bool {
		if users[i].Role == users[j].Role {
			return users[i].Name < users[j].Name
		}
		return users[i].Role < users[j].Role
	})
	return users, err
}

func (s *Store) DeleteUser(id string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("user_accounts"))
		if bucket.Get([]byte(id)) == nil {
			return model.ErrRequestNotFound
		}
		return bucket.Delete([]byte(id))
	})
}

func (s *Store) SetUserEnabled(id string, enabled bool) (model.UserAccount, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var user model.UserAccount
	err := s.db.Update(func(tx *bbolt.Tx) error {
		if err := get(tx, []byte("user_accounts"), id, &user); err != nil {
			return err
		}
		user.Enabled = enabled
		return put(tx, []byte("user_accounts"), id, user)
	})
	return user, err
}

func (s *Store) ReplaceGroupMembers(id string, members []string) (model.AgentGroup, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var group model.AgentGroup
	err := s.db.Update(func(tx *bbolt.Tx) error {
		if err := get(tx, []byte("agent_groups"), id, &group); err != nil {
			return model.ErrGroupNotFound
		}
		group.Members = append([]string(nil), members...)
		return put(tx, []byte("agent_groups"), id, group)
	})
	return group, err
}

func (s *Store) SetGroupActive(id string, active bool) (model.AgentGroup, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var group model.AgentGroup
	err := s.db.Update(func(tx *bbolt.Tx) error {
		if err := get(tx, []byte("agent_groups"), id, &group); err != nil {
			return model.ErrGroupNotFound
		}
		group.Active = active
		return put(tx, []byte("agent_groups"), id, group)
	})
	return group, err
}

func (s *Store) RequestsForAssignee(agentID string) ([]model.ServiceRequest, error) {
	items, err := s.ListRequests()
	if err != nil {
		return nil, err
	}
	result := make([]model.ServiceRequest, 0)
	for _, item := range items {
		if item.Assignee == agentID {
			result = append(result, item)
		}
	}
	return result, nil
}

func (s *Store) RequestsForGroup(groupID string) ([]model.ServiceRequest, error) {
	items, err := s.ListRequests()
	if err != nil {
		return nil, err
	}
	result := make([]model.ServiceRequest, 0)
	for _, item := range items {
		if item.GroupID == groupID {
			result = append(result, item)
		}
	}
	return result, nil
}

func (s *Store) RecentAudits(since time.Time) ([]model.AuditEvent, error) {
	items, err := s.ListAudits("")
	if err != nil {
		return nil, err
	}
	result := make([]model.AuditEvent, 0)
	for _, item := range items {
		if !item.CreatedAt.Before(since) {
			result = append(result, item)
		}
	}
	return result, nil
}

func (s *Store) DeleteAuditsBefore(cutoff time.Time) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	removed := 0
	err := s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("audit_events"))
		cursor := bucket.Cursor()
		for key, data := cursor.First(); key != nil; key, data = cursor.Next() {
			var event model.AuditEvent
			if err := decode(data, &event); err != nil {
				return err
			}
			if event.CreatedAt.Before(cutoff) {
				if err := cursor.Delete(); err != nil {
					return err
				}
				removed++
			}
		}
		return nil
	})
	return removed, err
}

func (s *Store) EntityCounts() (map[string]int, error) {
	result := make(map[string]int)
	for _, bucket := range []string{"service_requests", "agent_groups", "user_accounts", "audit_events", "dispatch_attempts"} {
		count, err := s.Count(bucket)
		if err != nil {
			return nil, err
		}
		result[bucket] = count
	}
	return result, nil
}
