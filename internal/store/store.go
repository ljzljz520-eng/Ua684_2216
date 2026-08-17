package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"go.etcd.io/bbolt"
	"service-request-dispatch/internal/model"
)

var bucketNames = [][]byte{[]byte("service_requests"), []byte("agent_groups"), []byte("user_accounts"), []byte("audit_events"), []byte("dispatch_attempts")}

type Store struct {
	db *bbolt.DB
	mu sync.RWMutex
}

func Open(path string) (*Store, error) {
	db, err := bbolt.Open(path, 0o600, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	if err := db.Update(func(tx *bbolt.Tx) error {
		for _, name := range bucketNames {
			if _, err := tx.CreateBucketIfNotExists(name); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize buckets: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}

func encode(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode entity: %w", err)
	}
	return data, nil
}

func decode(data []byte, target any) error {
	if len(data) == 0 {
		return model.ErrRequestNotFound
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode entity: %w", err)
	}
	return nil
}

func put(tx *bbolt.Tx, bucket []byte, key string, value any) error {
	if key == "" {
		return errors.New("empty key")
	}
	data, err := encode(value)
	if err != nil {
		return err
	}
	return tx.Bucket(bucket).Put([]byte(key), data)
}

func get(tx *bbolt.Tx, bucket []byte, key string, target any) error {
	value := tx.Bucket(bucket).Get([]byte(key))
	if value == nil {
		return model.ErrRequestNotFound
	}
	return decode(value, target)
}

func (s *Store) PutRequest(request model.ServiceRequest) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.db.Update(func(tx *bbolt.Tx) error { return put(tx, []byte("service_requests"), request.ID, request) })
}

func (s *Store) GetRequest(id string) (model.ServiceRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var request model.ServiceRequest
	err := s.db.View(func(tx *bbolt.Tx) error { return get(tx, []byte("service_requests"), id, &request) })
	return request, err
}

func (s *Store) DeleteRequest(id string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.db.Update(func(tx *bbolt.Tx) error { return tx.Bucket([]byte("service_requests")).Delete([]byte(id)) })
}

func (s *Store) ListRequests() ([]model.ServiceRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	requests := make([]model.ServiceRequest, 0)
	err := s.db.View(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte("service_requests")).ForEach(func(_, value []byte) error {
			var item model.ServiceRequest
			if err := decode(value, &item); err != nil {
				return err
			}
			requests = append(requests, item)
			return nil
		})
	})
	sort.Slice(requests, func(i, j int) bool { return requests[i].CreatedAt.Before(requests[j].CreatedAt) })
	return requests, err
}

func (s *Store) PutGroup(group model.AgentGroup) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.db.Update(func(tx *bbolt.Tx) error { return put(tx, []byte("agent_groups"), group.ID, group) })
}

func (s *Store) GetGroup(id string) (model.AgentGroup, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var group model.AgentGroup
	err := s.db.View(func(tx *bbolt.Tx) error { return get(tx, []byte("agent_groups"), id, &group) })
	if errors.Is(err, model.ErrRequestNotFound) {
		return group, model.ErrGroupNotFound
	}
	return group, err
}

func (s *Store) ListGroups() ([]model.AgentGroup, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	groups := make([]model.AgentGroup, 0)
	err := s.db.View(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte("agent_groups")).ForEach(func(_, value []byte) error {
			var item model.AgentGroup
			if err := decode(value, &item); err != nil {
				return err
			}
			groups = append(groups, item)
			return nil
		})
	})
	sort.Slice(groups, func(i, j int) bool { return groups[i].Name < groups[j].Name })
	return groups, err
}

func (s *Store) PutUser(user model.UserAccount) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.db.Update(func(tx *bbolt.Tx) error { return put(tx, []byte("user_accounts"), user.ID, user) })
}

func (s *Store) GetUser(id string) (model.UserAccount, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var user model.UserAccount
	err := s.db.View(func(tx *bbolt.Tx) error { return get(tx, []byte("user_accounts"), id, &user) })
	return user, err
}

func (s *Store) PutAttempt(attempt model.DispatchAttempt) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.db.Update(func(tx *bbolt.Tx) error { return put(tx, []byte("dispatch_attempts"), attempt.ID, attempt) })
}

func (s *Store) ListAttempts(requestID string) ([]model.DispatchAttempt, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]model.DispatchAttempt, 0)
	err := s.db.View(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte("dispatch_attempts")).ForEach(func(_, value []byte) error {
			var item model.DispatchAttempt
			if err := decode(value, &item); err != nil {
				return err
			}
			if requestID == "" || item.RequestID == requestID {
				items = append(items, item)
			}
			return nil
		})
	})
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return items, err
}

func (s *Store) PutAudit(event model.AuditEvent) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.db.Update(func(tx *bbolt.Tx) error { return put(tx, []byte("audit_events"), event.ID, event) })
}

func (s *Store) ListAudits(entityID string) ([]model.AuditEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]model.AuditEvent, 0)
	err := s.db.View(func(tx *bbolt.Tx) error {
		return tx.Bucket([]byte("audit_events")).ForEach(func(_, value []byte) error {
			var item model.AuditEvent
			if err := decode(value, &item); err != nil {
				return err
			}
			if entityID == "" || item.EntityID == entityID {
				items = append(items, item)
			}
			return nil
		})
	})
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return items, err
}

func (s *Store) Count(bucket string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return errors.New("bucket not found")
		}
		count = b.Stats().KeyN
		return nil
	})
	return count, err
}
