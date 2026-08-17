package model

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type RequestSummary struct {
	ID       string        `json:"id"`
	Title    string        `json:"title"`
	Status   RequestStatus `json:"status"`
	Group    string        `json:"group"`
	Owner    string        `json:"owner"`
	Priority string        `json:"priority"`
	Age      string        `json:"age"`
	Tags     string        `json:"tags"`
}

func SummarizeRequest(request ServiceRequest, now time.Time) RequestSummary {
	age := now.Sub(request.CreatedAt)
	if age < 0 {
		age = 0
	}
	return RequestSummary{ID: request.ID, Title: request.Subject, Status: request.Status, Group: fallback(request.GroupID, "unassigned"), Owner: fallback(request.Assignee, "unassigned"), Priority: PriorityLabel(request.Priority), Age: FormatDuration(age), Tags: strings.Join(request.Tags, ", ")}
}

func SummarizeRequests(requests []ServiceRequest, now time.Time) []RequestSummary {
	result := make([]RequestSummary, 0, len(requests))
	for _, request := range requests {
		result = append(result, SummarizeRequest(request, now))
	}
	return result
}

func FormatDuration(value time.Duration) string {
	if value < time.Minute {
		return "less than a minute"
	}
	if value < time.Hour {
		return fmt.Sprintf("%d minutes", int(value.Minutes()))
	}
	if value < 24*time.Hour {
		return fmt.Sprintf("%d hours", int(value.Hours()))
	}
	return fmt.Sprintf("%d days", int(value.Hours()/24))
}

func StatusLabel(status RequestStatus) string {
	switch status {
	case StatusQueued:
		return "Queued"
	case StatusAssigned:
		return "Assigned"
	case StatusPending:
		return "Waiting"
	case StatusResolved:
		return "Resolved"
	case StatusRejected:
		return "Rejected"
	default:
		return "Unknown"
	}
}

func RoleLabel(role UserRole) string {
	switch role {
	case RoleAgent:
		return "Support agent"
	case RoleManager:
		return "Service manager"
	case RoleAuditor:
		return "Audit reviewer"
	case RoleAdmin:
		return "Administrator"
	default:
		return "Unknown role"
	}
}

func SortTags(tags []string) []string {
	result := NormalizeTags(tags)
	sort.Strings(result)
	return result
}

func ContainsAllTags(have, wanted []string) bool {
	set := make(map[string]bool)
	for _, tag := range have {
		set[strings.ToLower(strings.TrimSpace(tag))] = true
	}
	for _, tag := range wanted {
		if !set[strings.ToLower(strings.TrimSpace(tag))] {
			return false
		}
	}
	return true
}

func DisplayName(user UserAccount) string {
	if strings.TrimSpace(user.Name) != "" {
		return strings.TrimSpace(user.Name)
	}
	if strings.TrimSpace(user.Email) != "" {
		return strings.TrimSpace(user.Email)
	}
	return user.ID
}

func GroupDisplayName(group AgentGroup) string {
	if strings.TrimSpace(group.Name) == "" {
		return group.ID
	}
	if group.Active {
		return group.Name
	}
	return group.Name + " (inactive)"
}

func AttemptLabel(attempt DispatchAttempt) string {
	if attempt.Outcome == "assigned" {
		return fmt.Sprintf("Assigned to %s in %s", attempt.AgentID, attempt.GroupID)
	}
	if attempt.Reason != "" {
		return fmt.Sprintf("%s: %s", attempt.Outcome, attempt.Reason)
	}
	return attempt.Outcome
}

func AuditLabel(event AuditEvent) string {
	actor := fallback(event.ActorID, "system")
	if event.Details == "" {
		return fmt.Sprintf("%s performed %s", actor, event.Action)
	}
	return fmt.Sprintf("%s performed %s: %s", actor, event.Action, event.Details)
}

func fallback(value, replacement string) string {
	if strings.TrimSpace(value) == "" {
		return replacement
	}
	return value
}
