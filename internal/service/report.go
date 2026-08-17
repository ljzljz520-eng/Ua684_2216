package service

import (
	"sort"
	"strings"
	"time"

	"service-request-dispatch/internal/model"
)

type Dashboard struct {
	Total           int                         `json:"total"`
	Open            int                         `json:"open"`
	ByStatus        map[model.RequestStatus]int `json:"by_status"`
	ByGroup         map[string]int              `json:"by_group"`
	AveragePriority float64                     `json:"average_priority"`
	Oldest          *model.ServiceRequest       `json:"oldest,omitempty"`
}

func BuildDashboard(items []model.ServiceRequest) Dashboard {
	result := Dashboard{ByStatus: make(map[model.RequestStatus]int), ByGroup: make(map[string]int)}
	result.Total = len(items)
	var totalPriority int
	for index := range items {
		item := items[index]
		result.ByStatus[item.Status]++
		result.ByGroup[item.GroupID]++
		if item.IsOpen() {
			result.Open++
		}
		totalPriority += item.Priority
		if result.Oldest == nil || item.CreatedAt.Before(result.Oldest.CreatedAt) {
			copy := item
			result.Oldest = &copy
		}
	}
	if result.Total > 0 {
		result.AveragePriority = float64(totalPriority) / float64(result.Total)
	}
	return result
}

func SortForManager(items []model.ServiceRequest, newestFirst bool) []model.ServiceRequest {
	result := append([]model.ServiceRequest(nil), items...)
	sort.SliceStable(result, func(i, j int) bool {
		if newestFirst {
			return result[i].CreatedAt.After(result[j].CreatedAt)
		}
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result
}

func Search(items []model.ServiceRequest, phrase string) []model.ServiceRequest {
	needle := strings.ToLower(strings.TrimSpace(phrase))
	if needle == "" {
		return append([]model.ServiceRequest(nil), items...)
	}
	result := make([]model.ServiceRequest, 0)
	for _, item := range items {
		joined := strings.ToLower(item.Subject + " " + item.Description + " " + item.Customer + " " + strings.Join(item.Tags, " "))
		if strings.Contains(joined, needle) {
			result = append(result, item)
		}
	}
	return result
}

func TimeRange(items []model.ServiceRequest, from, to time.Time) []model.ServiceRequest {
	result := make([]model.ServiceRequest, 0)
	for _, item := range items {
		if !from.IsZero() && item.CreatedAt.Before(from) {
			continue
		}
		if !to.IsZero() && item.CreatedAt.After(to) {
			continue
		}
		result = append(result, item)
	}
	return result
}
