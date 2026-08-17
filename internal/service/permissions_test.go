package service

import (
	"testing"

	"service-request-dispatch/internal/model"
)

func TestPermissionMatrix(t *testing.T) {
	if !PermissionFor(model.RoleManager, PermissionDispatch) {
		t.Fatal("manager dispatch permission missing")
	}
	if PermissionFor(model.RoleAgent, PermissionExport) {
		t.Fatal("agent export permission should be denied")
	}
	if _, err := ParsePermission("request:read"); err != nil {
		t.Fatal(err)
	}
	if _, err := ParsePermission("bad"); err == nil {
		t.Fatal("expected unknown permission")
	}
}
