package filter

import (
	"sort"
	"strings"
	"time"

	"service-request-dispatch/internal/model"
)

func Match(request model.ServiceRequest, query model.RequestFilter) bool {
	if query.Status != "" && request.Status != query.Status {
		return false
	}
	if query.GroupID != "" && request.GroupID != query.GroupID {
		return false
	}
	if query.Assignee != "" && request.Assignee != query.Assignee {
		return false
	}
	if query.Priority > 0 && request.Priority != query.Priority {
		return false
	}
	if !query.From.IsZero() && request.CreatedAt.Before(query.From) {
		return false
	}
	if !query.To.IsZero() && request.CreatedAt.After(query.To) {
		return false
	}
	if query.Tag != "" && !containsTag(request.Tags, query.Tag) {
		return false
	}
	return true
}

func containsTag(tags []string, wanted string) bool {
	for _, tag := range tags {
		if strings.EqualFold(tag, wanted) {
			return true
		}
	}
	return false
}

func Apply(requests []model.ServiceRequest, query model.RequestFilter) []model.ServiceRequest {
	result := make([]model.ServiceRequest, 0, len(requests))
	for _, request := range requests {
		if Match(request, query) {
			result = append(result, request)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Priority == result[j].Priority {
			return result[i].CreatedAt.Before(result[j].CreatedAt)
		}
		return result[i].Priority > result[j].Priority
	})
	return result
}

func Window(from, to time.Time) model.RequestFilter { return model.RequestFilter{From: from, To: to} }

func StatusCount(requests []model.ServiceRequest) map[model.RequestStatus]int {
	counts := make(map[model.RequestStatus]int)
	for _, request := range requests {
		counts[request.Status]++
	}
	return counts
}
