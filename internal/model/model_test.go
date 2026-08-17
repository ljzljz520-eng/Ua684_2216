package model

import "testing"

func TestEntityValidationAndLabels(t *testing.T) {
	if err := (ServiceRequest{ID: "r", Subject: "s", Customer: "c", Description: "valid description", Priority: 3}).Validate(); err != nil {
		t.Fatal(err)
	}
	if err := (UserAccount{ID: "u", Name: "User", Email: "u@example.test", Role: RoleAgent, Enabled: true}).Validate(); err != nil {
		t.Fatal(err)
	}
	if PriorityLabel(5) != "urgent" || PriorityLabel(0) != "unknown" {
		t.Fatal("priority labels incorrect")
	}
	tags := NormalizeTags([]string{" Billing ", "billing", "", "Support"})
	if len(tags) != 2 || tags[0] != "billing" {
		t.Fatalf("tags=%v", tags)
	}
}

func TestTransitionTable(t *testing.T) {
	if !CanTransition(StatusQueued, StatusAssigned) {
		t.Fatal("queued should assign")
	}
	if CanTransition(StatusResolved, StatusAssigned) {
		t.Fatal("resolved should be terminal")
	}
	if NormalizeStatus("unknown") != StatusQueued {
		t.Fatal("unknown should normalize")
	}
}
