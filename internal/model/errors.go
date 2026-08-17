package model

import "errors"

var (
	ErrMissingRequestID    = errors.New("request id is required")
	ErrMissingSubject      = errors.New("subject is required")
	ErrMissingCustomer     = errors.New("customer is required")
	ErrDescriptionTooShort = errors.New("description must contain at least five characters")
	ErrInvalidPriority     = errors.New("priority must be between one and five")
	ErrGroupNotFound       = errors.New("agent group not found")
	ErrGroupInactive       = errors.New("agent group is inactive")
	ErrNoAvailableAgent    = errors.New("no available agent in group")
	ErrRequestNotFound     = errors.New("service request not found")
	ErrPermissionDenied    = errors.New("permission denied")
	ErrInvalidTransition   = errors.New("invalid request status transition")
	ErrAlreadyExists       = errors.New("entity already exists")
)
