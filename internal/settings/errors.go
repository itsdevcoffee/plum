package settings

import "errors"

// Errors for the settings package
var (
	// ErrInvalidScope is returned when an unknown scope is provided
	ErrInvalidScope = errors.New("invalid scope: must be managed, user, project, or local")

	// ErrManagedReadOnly is returned when attempting to write to managed scope
	ErrManagedReadOnly = errors.New("managed scope is read-only")
)
