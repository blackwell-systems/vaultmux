package vaultmux

import (
	"errors"
	"fmt"
)

// BackendError wraps errors with backend context.
type BackendError struct {
	Backend string
	Op      string // Operation: "get", "create", "delete", etc.
	Item    string // Item name (if applicable)
	Err     error
}

// Error returns the error message.
func (e *BackendError) Error() string {
	if e.Item != "" {
		return fmt.Sprintf("%s: %s %q: %v", e.Backend, e.Op, e.Item, e.Err)
	}
	return fmt.Sprintf("%s: %s: %v", e.Backend, e.Op, e.Err)
}

// Unwrap returns the underlying error.
func (e *BackendError) Unwrap() error {
	return e.Err
}

// Is checks if the wrapped error matches the target error.
// This allows errors.Is() to work through BackendError wrappers.
func (e *BackendError) Is(target error) bool {
	return errors.Is(e.Err, target)
}

// WrapError wraps an error with backend context.
func WrapError(backend, op, item string, err error) error {
	if err == nil {
		return nil
	}
	return &BackendError{
		Backend: backend,
		Op:      op,
		Item:    item,
		Err:     err,
	}
}
