package filter

import (
	"testing"
	"time"

	"service-request-dispatch/internal/model"
)

func TestApplyFilterSortsPriorityAndMatchesTags(t *testing.T) {
	items := []model.ServiceRequest{{ID: "low", Status: model.StatusQueued, GroupID: "g", Priority: 1, CreatedAt: time.Unix(2, 0), Tags: []string{"billing"}}, {ID: "high", Status: model.StatusQueued, GroupID: "g", Priority: 5, CreatedAt: time.Unix(3, 0), Tags: []string{"Billing"}}, {ID: "other", Status: model.StatusAssigned, GroupID: "x", Priority: 5}}
	got := Apply(items, model.RequestFilter{Status: model.StatusQueued, GroupID: "g", Tag: "billing"})
	if len(got) != 2 || got[0].ID != "high" {
		t.Fatalf("filtered=%#v", got)
	}
}

func TestStatusCount(t *testing.T) {
	counts := StatusCount([]model.ServiceRequest{{Status: model.StatusQueued}, {Status: model.StatusQueued}, {Status: model.StatusResolved}})
	if counts[model.StatusQueued] != 2 || counts[model.StatusResolved] != 1 {
		t.Fatalf("counts=%v", counts)
	}
}
