package model

import (
	"fmt"
	"regexp"
	"strings"
)

var emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

func (u UserAccount) Validate() error {
	if strings.TrimSpace(u.ID) == "" {
		return fmt.Errorf("user id is required")
	}
	if strings.TrimSpace(u.Name) == "" {
		return fmt.Errorf("user name is required")
	}
	if !emailPattern.MatchString(u.Email) {
		return fmt.Errorf("user email is invalid")
	}
	if u.Role == "" {
		return fmt.Errorf("user role is required")
	}
	if !u.Enabled {
		return fmt.Errorf("user must be enabled")
	}
	return nil
}

func (g AgentGroup) Validate() error {
	if strings.TrimSpace(g.ID) == "" {
		return fmt.Errorf("group id is required")
	}
	if strings.TrimSpace(g.Name) == "" {
		return fmt.Errorf("group name is required")
	}
	if !g.Active {
		return ErrGroupInactive
	}
	if len(g.Members) == 0 {
		return ErrNoAvailableAgent
	}
	seen := make(map[string]struct{}, len(g.Members))
	for _, member := range g.Members {
		if member == "" {
			return fmt.Errorf("group member is empty")
		}
		if _, exists := seen[member]; exists {
			return fmt.Errorf("duplicate group member %s", member)
		}
		seen[member] = struct{}{}
	}
	return nil
}

func (a DispatchAttempt) Validate() error {
	if a.ID == "" || a.RequestID == "" {
		return fmt.Errorf("attempt identity is required")
	}
	if a.Outcome == "" {
		return fmt.Errorf("attempt outcome is required")
	}
	return nil
}

func (e AuditEvent) Validate() error {
	if e.ID == "" || e.EntityID == "" {
		return fmt.Errorf("audit identity is required")
	}
	if e.Action == "" {
		return fmt.Errorf("audit action is required")
	}
	return nil
}

func NormalizeTags(tags []string) []string {
	result := make([]string, 0, len(tags))
	seen := make(map[string]bool)
	for _, tag := range tags {
		clean := strings.ToLower(strings.TrimSpace(tag))
		if clean == "" || seen[clean] {
			continue
		}
		seen[clean] = true
		result = append(result, clean)
	}
	return result
}

func PriorityLabel(priority int) string {
	switch priority {
	case 5:
		return "urgent"
	case 4:
		return "high"
	case 3:
		return "normal"
	case 2:
		return "low"
	case 1:
		return "minimal"
	default:
		return "unknown"
	}
}
