package service

import (
	"fmt"
	"strings"

	"service-request-dispatch/internal/model"
)

type Permission string

const (
	PermissionCreate   Permission = "request:create"
	PermissionRead     Permission = "request:read"
	PermissionDispatch Permission = "request:dispatch"
	PermissionResolve  Permission = "request:resolve"
	PermissionExport   Permission = "request:export"
	PermissionAudit    Permission = "audit:read"
)

var rolePermissions = map[model.UserRole]map[Permission]bool{
	model.RoleAgent:   {PermissionRead: true, PermissionResolve: true},
	model.RoleManager: {PermissionCreate: true, PermissionRead: true, PermissionDispatch: true, PermissionResolve: true, PermissionExport: true},
	model.RoleAuditor: {PermissionRead: true, PermissionExport: true, PermissionAudit: true},
	model.RoleAdmin:   {PermissionCreate: true, PermissionRead: true, PermissionDispatch: true, PermissionResolve: true, PermissionExport: true, PermissionAudit: true},
}

func PermissionFor(role model.UserRole, permission Permission) bool {
	return rolePermissions[role][permission]
}

func (s *Service) Authorize(userID string, permission Permission) error {
	user, err := s.GetUser(userID)
	if err != nil {
		return err
	}
	if !user.Enabled || !PermissionFor(user.Role, permission) {
		return model.ErrPermissionDenied
	}
	return nil
}

func RequireAnyPermission(s *Service, userID string, permissions ...Permission) error {
	if len(permissions) == 0 {
		return fmt.Errorf("at least one permission is required")
	}
	for _, permission := range permissions {
		if err := s.Authorize(userID, permission); err == nil {
			return nil
		}
	}
	return model.ErrPermissionDenied
}

func ParsePermission(value string) (Permission, error) {
	clean := Permission(strings.TrimSpace(value))
	for _, candidate := range []Permission{PermissionCreate, PermissionRead, PermissionDispatch, PermissionResolve, PermissionExport, PermissionAudit} {
		if clean == candidate {
			return clean, nil
		}
	}
	return "", fmt.Errorf("unknown permission %q", value)
}
