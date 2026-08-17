package service

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"service-request-dispatch/internal/model"
)

type GroupLoad struct {
	GroupID      string  `json:"group_id"`
	Open         int     `json:"open"`
	Assigned     int     `json:"assigned"`
	Members      int     `json:"members"`
	LoadPerAgent float64 `json:"load_per_agent"`
}

type AgentLoad struct {
	AgentID  string `json:"agent_id"`
	Assigned int    `json:"assigned"`
	Pending  int    `json:"pending"`
	Resolved int    `json:"resolved"`
}

type RetentionResult struct {
	Cutoff        time.Time `json:"cutoff"`
	RemovedAudits int       `json:"removed_audits"`
}

func (s *Service) ListUsers() ([]model.UserAccount, error) { return s.store.ListUsers() }

func (s *Service) DisableUser(actor, userID string) (model.UserAccount, error) {
	if err := s.Authorize(actor, PermissionAudit); err != nil {
		return model.UserAccount{}, err
	}
	user, err := s.store.SetUserEnabled(userID, false)
	if err != nil {
		return user, err
	}
	if err := s.audit.Record(actor, "user.disabled", "user", userID, "account disabled", s.now()); err != nil {
		return user, err
	}
	return user, nil
}

func (s *Service) EnableUser(actor, userID string) (model.UserAccount, error) {
	if err := s.Authorize(actor, PermissionAudit); err != nil {
		return model.UserAccount{}, err
	}
	user, err := s.store.SetUserEnabled(userID, true)
	if err != nil {
		return user, err
	}
	if err := s.audit.Record(actor, "user.enabled", "user", userID, "account enabled", s.now()); err != nil {
		return user, err
	}
	return user, nil
}

func (s *Service) ReplaceGroupMembers(actor, groupID string, members []string) (model.AgentGroup, error) {
	if err := s.Authorize(actor, PermissionDispatch); err != nil {
		return model.AgentGroup{}, err
	}
	if len(members) == 0 {
		return model.AgentGroup{}, model.ErrNoAvailableAgent
	}
	seen := make(map[string]bool)
	for _, member := range members {
		if seen[member] {
			return model.AgentGroup{}, fmt.Errorf("duplicate group member %s", member)
		}
		seen[member] = true
		user, err := s.GetUser(member)
		if err != nil {
			return model.AgentGroup{}, err
		}
		if !user.Enabled || user.Role != model.RoleAgent {
			return model.AgentGroup{}, fmt.Errorf("member %s is not available", member)
		}
	}
	group, err := s.store.ReplaceGroupMembers(groupID, members)
	if err != nil {
		return group, err
	}
	detail := strings.Join(members, ",")
	if err := s.audit.Record(actor, "group.members_replaced", "group", groupID, detail, s.now()); err != nil {
		return group, err
	}
	return group, nil
}

func (s *Service) SetGroupActive(actor, groupID string, active bool) (model.AgentGroup, error) {
	if err := s.Authorize(actor, PermissionDispatch); err != nil {
		return model.AgentGroup{}, err
	}
	group, err := s.store.SetGroupActive(groupID, active)
	if err != nil {
		return group, err
	}
	action := "group.disabled"
	if active {
		action = "group.enabled"
	}
	if err := s.audit.Record(actor, action, "group", groupID, fmt.Sprintf("active=%t", active), s.now()); err != nil {
		return group, err
	}
	return group, nil
}

func (s *Service) GroupLoads() ([]GroupLoad, error) {
	groups, err := s.ListGroups()
	if err != nil {
		return nil, err
	}
	items, err := s.store.ListRequests()
	if err != nil {
		return nil, err
	}
	byGroup := make(map[string][]model.ServiceRequest)
	for _, item := range items {
		byGroup[item.GroupID] = append(byGroup[item.GroupID], item)
	}
	loads := make([]GroupLoad, 0, len(groups))
	for _, group := range groups {
		load := GroupLoad{GroupID: group.ID, Members: len(group.Members)}
		for _, item := range byGroup[group.ID] {
			if item.IsOpen() {
				load.Open++
			}
			if item.Status == model.StatusAssigned {
				load.Assigned++
			}
		}
		if load.Members > 0 {
			load.LoadPerAgent = float64(load.Open) / float64(load.Members)
		}
		loads = append(loads, load)
	}
	sort.Slice(loads, func(i, j int) bool {
		if loads[i].LoadPerAgent == loads[j].LoadPerAgent {
			return loads[i].GroupID < loads[j].GroupID
		}
		return loads[i].LoadPerAgent > loads[j].LoadPerAgent
	})
	return loads, nil
}

func (s *Service) AgentLoads() ([]AgentLoad, error) {
	users, err := s.ListUsers()
	if err != nil {
		return nil, err
	}
	result := make([]AgentLoad, 0)
	for _, user := range users {
		if user.Role != model.RoleAgent {
			continue
		}
		items, listErr := s.store.RequestsForAssignee(user.ID)
		if listErr != nil {
			return nil, listErr
		}
		load := AgentLoad{AgentID: user.ID}
		for _, item := range items {
			switch item.Status {
			case model.StatusAssigned:
				load.Assigned++
			case model.StatusPending:
				load.Pending++
			case model.StatusResolved:
				load.Resolved++
			}
		}
		result = append(result, load)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].AgentID < result[j].AgentID })
	return result, nil
}

func (s *Service) ApplyAuditRetention(actor string, days int) (RetentionResult, error) {
	if err := s.Authorize(actor, PermissionAudit); err != nil {
		return RetentionResult{}, err
	}
	if days < 1 {
		return RetentionResult{}, fmt.Errorf("retention days must be positive")
	}
	cutoff := s.now().Add(-time.Duration(days) * 24 * time.Hour)
	removed, err := s.store.DeleteAuditsBefore(cutoff)
	if err != nil {
		return RetentionResult{}, err
	}
	result := RetentionResult{Cutoff: cutoff, RemovedAudits: removed}
	if err := s.audit.Record(actor, "audit.retention_applied", "audit", "retention", fmt.Sprintf("removed=%d", removed), s.now()); err != nil {
		return result, err
	}
	return result, nil
}

func (s *Service) EntityCounts() (map[string]int, error) { return s.store.EntityCounts() }

func (s *Service) RecentAudits(since time.Time) ([]model.AuditEvent, error) {
	return s.store.RecentAudits(since)
}
