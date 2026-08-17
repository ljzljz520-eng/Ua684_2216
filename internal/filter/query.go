package filter

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"service-request-dispatch/internal/model"
)

type Page struct {
	Number int
	Size   int
	Total  int
	Items  []model.ServiceRequest
}

type Facets struct {
	Status map[model.RequestStatus]int
	Groups map[string]int
	Agents map[string]int
	Tags   map[string]int
}

func Parse(values url.Values) (model.RequestFilter, error) {
	result := model.RequestFilter{Status: model.RequestStatus(strings.TrimSpace(values.Get("status"))), GroupID: strings.TrimSpace(values.Get("group")), Assignee: strings.TrimSpace(values.Get("assignee")), Tag: strings.TrimSpace(values.Get("tag"))}
	if raw := values.Get("priority"); raw != "" {
		priority, err := strconv.Atoi(raw)
		if err != nil || priority < 1 || priority > 5 {
			return result, fmt.Errorf("invalid priority")
		}
		result.Priority = priority
	}
	if raw := values.Get("from"); raw != "" {
		value, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return result, fmt.Errorf("invalid from time")
		}
		result.From = value
	}
	if raw := values.Get("to"); raw != "" {
		value, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return result, fmt.Errorf("invalid to time")
		}
		result.To = value
	}
	if !result.From.IsZero() && !result.To.IsZero() && result.From.After(result.To) {
		return result, fmt.Errorf("from time is after to time")
	}
	return result, nil
}

func Paginate(items []model.ServiceRequest, number, size int) Page {
	if number < 1 {
		number = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 200 {
		size = 200
	}
	start := (number - 1) * size
	if start >= len(items) {
		return Page{Number: number, Size: size, Total: len(items), Items: []model.ServiceRequest{}}
	}
	end := start + size
	if end > len(items) {
		end = len(items)
	}
	return Page{Number: number, Size: size, Total: len(items), Items: append([]model.ServiceRequest(nil), items[start:end]...)}
}

func BuildFacets(items []model.ServiceRequest) Facets {
	result := Facets{Status: make(map[model.RequestStatus]int), Groups: make(map[string]int), Agents: make(map[string]int), Tags: make(map[string]int)}
	for _, item := range items {
		result.Status[item.Status]++
		if item.GroupID != "" {
			result.Groups[item.GroupID]++
		}
		if item.Assignee != "" {
			result.Agents[item.Assignee]++
		}
		for _, tag := range item.Tags {
			result.Tags[strings.ToLower(tag)]++
		}
	}
	return result
}

func UniqueGroups(items []model.ServiceRequest) []string {
	set := make(map[string]bool)
	for _, item := range items {
		if item.GroupID != "" {
			set[item.GroupID] = true
		}
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func OpenOnly(items []model.ServiceRequest) []model.ServiceRequest {
	result := make([]model.ServiceRequest, 0)
	for _, item := range items {
		if item.IsOpen() {
			result = append(result, item)
		}
	}
	return result
}

func TerminalOnly(items []model.ServiceRequest) []model.ServiceRequest {
	result := make([]model.ServiceRequest, 0)
	for _, item := range items {
		if item.IsTerminal() {
			result = append(result, item)
		}
	}
	return result
}

func ExcludeGroup(items []model.ServiceRequest, groupID string) []model.ServiceRequest {
	result := make([]model.ServiceRequest, 0)
	for _, item := range items {
		if item.GroupID != groupID {
			result = append(result, item)
		}
	}
	return result
}
