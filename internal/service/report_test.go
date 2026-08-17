package service

import (
	"testing"
	"time"

	"service-request-dispatch/internal/model"
)

func TestBuildDashboardAndSearch(t *testing.T) {
	items := []model.ServiceRequest{{ID: "r1", Subject: "Billing delay", Description: "invoice", Customer: "one", Status: model.StatusQueued, GroupID: "g", Priority: 5, CreatedAt: time.Unix(5, 0)}, {ID: "r2", Subject: "Login", Description: "password", Customer: "two", Status: model.StatusResolved, GroupID: "h", Priority: 1, CreatedAt: time.Unix(1, 0)}}
	dashboard := BuildDashboard(items)
	if dashboard.Total != 2 || dashboard.Open != 1 || dashboard.ByGroup["g"] != 1 {
		t.Fatalf("dashboard=%#v", dashboard)
	}
	if len(Search(items, "billing")) != 1 {
		t.Fatal("search failed")
	}
	if len(TimeRange(items, time.Unix(2, 0), time.Time{})) != 1 {
		t.Fatal("time range failed")
	}
}
