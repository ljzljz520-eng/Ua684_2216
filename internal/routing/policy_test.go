package routing

import (
	"testing"
	"time"

	"service-request-dispatch/internal/model"
)

func TestPolicyDecidesByPriorityAndTag(t *testing.T) {
	policy, err := NewPolicy([]Rule{{ID: "urgent-billing", Name: "Urgent billing", GroupID: "billing", RequiredTags: []string{"billing"}, MinimumPriority: 4, Enabled: true, Order: 1}}, "general")
	if err != nil {
		t.Fatal(err)
	}
	decision := policy.Decide(model.ServiceRequest{Customer: "c", Priority: 5, Tags: []string{"Billing"}}, time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))
	if !decision.Matched || decision.GroupID != "billing" {
		t.Fatalf("decision=%#v", decision)
	}
	fallback := policy.Decide(model.ServiceRequest{Priority: 1}, time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))
	if fallback.Matched || fallback.GroupID != "general" {
		t.Fatalf("fallback=%#v", fallback)
	}
}
