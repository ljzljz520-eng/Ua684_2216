package routing

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"service-request-dispatch/internal/model"
)

type Rule struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	GroupID         string   `json:"group_id"`
	Customer        string   `json:"customer"`
	RequiredTags    []string `json:"required_tags"`
	MinimumPriority int      `json:"minimum_priority"`
	MaximumPriority int      `json:"maximum_priority"`
	StartHour       int      `json:"start_hour"`
	EndHour         int      `json:"end_hour"`
	Enabled         bool     `json:"enabled"`
	Order           int      `json:"order"`
}

type Decision struct {
	GroupID string `json:"group_id"`
	RuleID  string `json:"rule_id"`
	Reason  string `json:"reason"`
	Matched bool   `json:"matched"`
}

type Policy struct {
	rules    []Rule
	fallback string
}

func NewPolicy(rules []Rule, fallback string) (*Policy, error) {
	if strings.TrimSpace(fallback) == "" {
		return nil, fmt.Errorf("fallback group is required")
	}
	copyRules := append([]Rule(nil), rules...)
	seen := make(map[string]bool)
	for _, rule := range copyRules {
		if err := rule.Validate(); err != nil {
			return nil, fmt.Errorf("rule %s: %w", rule.ID, err)
		}
		if seen[rule.ID] {
			return nil, fmt.Errorf("duplicate routing rule %s", rule.ID)
		}
		seen[rule.ID] = true
	}
	sort.SliceStable(copyRules, func(i, j int) bool {
		if copyRules[i].Order == copyRules[j].Order {
			return copyRules[i].ID < copyRules[j].ID
		}
		return copyRules[i].Order < copyRules[j].Order
	})
	return &Policy{rules: copyRules, fallback: fallback}, nil
}

func (r Rule) Validate() error {
	if strings.TrimSpace(r.ID) == "" {
		return fmt.Errorf("id is required")
	}
	if strings.TrimSpace(r.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(r.GroupID) == "" {
		return fmt.Errorf("group is required")
	}
	if r.MinimumPriority < 0 || r.MinimumPriority > 5 {
		return fmt.Errorf("minimum priority is invalid")
	}
	if r.MaximumPriority < 0 || r.MaximumPriority > 5 {
		return fmt.Errorf("maximum priority is invalid")
	}
	if r.MinimumPriority > 0 && r.MaximumPriority > 0 && r.MinimumPriority > r.MaximumPriority {
		return fmt.Errorf("priority range is invalid")
	}
	if r.StartHour < 0 || r.StartHour > 23 {
		return fmt.Errorf("start hour is invalid")
	}
	if r.EndHour < 0 || r.EndHour > 24 {
		return fmt.Errorf("end hour is invalid")
	}
	return nil
}

func (p *Policy) Decide(request model.ServiceRequest, at time.Time) Decision {
	for _, rule := range p.rules {
		matched, reason := rule.Match(request, at)
		if matched {
			return Decision{GroupID: rule.GroupID, RuleID: rule.ID, Reason: reason, Matched: true}
		}
	}
	return Decision{GroupID: p.fallback, Reason: "no enabled rule matched", Matched: false}
}

func (r Rule) Match(request model.ServiceRequest, at time.Time) (bool, string) {
	if !r.Enabled {
		return false, "rule disabled"
	}
	if r.Customer != "" && !strings.EqualFold(r.Customer, request.Customer) {
		return false, "customer mismatch"
	}
	if r.MinimumPriority > 0 && request.Priority < r.MinimumPriority {
		return false, "priority below minimum"
	}
	if r.MaximumPriority > 0 && request.Priority > r.MaximumPriority {
		return false, "priority above maximum"
	}
	if r.StartHour != 0 || r.EndHour != 0 {
		hour := at.Hour()
		if r.StartHour < r.EndHour {
			if hour < r.StartHour || hour >= r.EndHour {
				return false, "outside routing hours"
			}
		} else if r.StartHour > r.EndHour {
			if hour < r.StartHour && hour >= r.EndHour {
				return false, "outside overnight hours"
			}
		}
	}
	requestTags := make(map[string]bool)
	for _, tag := range request.Tags {
		requestTags[strings.ToLower(strings.TrimSpace(tag))] = true
	}
	for _, required := range r.RequiredTags {
		if !requestTags[strings.ToLower(strings.TrimSpace(required))] {
			return false, "required tag missing"
		}
	}
	return true, "rule matched"
}

func (p *Policy) Rules() []Rule { return append([]Rule(nil), p.rules...) }

func (p *Policy) EnabledRules() []Rule {
	result := make([]Rule, 0)
	for _, rule := range p.rules {
		if rule.Enabled {
			result = append(result, rule)
		}
	}
	return result
}

func (p *Policy) RulesForGroup(groupID string) []Rule {
	result := make([]Rule, 0)
	for _, rule := range p.rules {
		if rule.GroupID == groupID {
			result = append(result, rule)
		}
	}
	return result
}

func (p *Policy) Replace(rule Rule) error {
	if err := rule.Validate(); err != nil {
		return err
	}
	found := false
	for index := range p.rules {
		if p.rules[index].ID == rule.ID {
			p.rules[index] = rule
			found = true
			break
		}
	}
	if !found {
		p.rules = append(p.rules, rule)
	}
	sort.SliceStable(p.rules, func(i, j int) bool { return p.rules[i].Order < p.rules[j].Order })
	return nil
}

func (p *Policy) Remove(id string) bool {
	for index := range p.rules {
		if p.rules[index].ID == id {
			p.rules = append(p.rules[:index], p.rules[index+1:]...)
			return true
		}
	}
	return false
}

func (p *Policy) SetFallback(groupID string) error {
	if strings.TrimSpace(groupID) == "" {
		return fmt.Errorf("fallback group is required")
	}
	p.fallback = groupID
	return nil
}

func (p *Policy) Fallback() string { return p.fallback }

func Explain(decision Decision) string {
	if decision.Matched {
		return fmt.Sprintf("routed to %s by rule %s: %s", decision.GroupID, decision.RuleID, decision.Reason)
	}
	return fmt.Sprintf("routed to fallback %s: %s", decision.GroupID, decision.Reason)
}

func ValidateGroups(policy *Policy, groups []model.AgentGroup) error {
	active := make(map[string]bool)
	for _, group := range groups {
		active[group.ID] = group.Active
	}
	if !active[policy.fallback] {
		return fmt.Errorf("fallback group is unavailable")
	}
	for _, rule := range policy.rules {
		if rule.Enabled && !active[rule.GroupID] {
			return fmt.Errorf("rule %s targets unavailable group", rule.ID)
		}
	}
	return nil
}

func GroupCoverage(policy *Policy, groups []model.AgentGroup) map[string]int {
	result := make(map[string]int)
	for _, group := range groups {
		result[group.ID] = 0
	}
	for _, rule := range policy.rules {
		if rule.Enabled {
			result[rule.GroupID]++
		}
	}
	result[policy.fallback]++
	return result
}

func MergeRules(base, overrides []Rule) []Rule {
	byID := make(map[string]Rule)
	for _, rule := range base {
		byID[rule.ID] = rule
	}
	for _, rule := range overrides {
		byID[rule.ID] = rule
	}
	result := make([]Rule, 0, len(byID))
	for _, rule := range byID {
		result = append(result, rule)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Order == result[j].Order {
			return result[i].ID < result[j].ID
		}
		return result[i].Order < result[j].Order
	})
	return result
}

func CloneRule(rule Rule) Rule {
	copy := rule
	copy.RequiredTags = append([]string(nil), rule.RequiredTags...)
	return copy
}
