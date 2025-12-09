package vaultmux

import (
	"errors"
	"testing"
)

func TestBackendError_Is(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		target error
		want   bool
	}{
		{
			name:   "wrapped ErrNotFound matches",
			err:    WrapError("test", "get", "item1", ErrNotFound),
			target: ErrNotFound,
			want:   true,
		},
		{
			name:   "wrapped ErrAlreadyExists matches",
			err:    WrapError("test", "create", "item2", ErrAlreadyExists),
			target: ErrAlreadyExists,
			want:   true,
		},
		{
			name:   "wrapped ErrNotAuthenticated matches",
			err:    WrapError("test", "auth", "", ErrNotAuthenticated),
			target: ErrNotAuthenticated,
			want:   true,
		},
		{
			name:   "wrapped generic error does not match sentinel",
			err:    WrapError("test", "get", "item3", errors.New("some error")),
			target: ErrNotFound,
			want:   false,
		},
		{
			name:   "double wrapped error matches",
			err:    WrapError("outer", "op", "item", WrapError("inner", "get", "item", ErrNotFound)),
			target: ErrNotFound,
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := errors.Is(tt.err, tt.target)
			if got != tt.want {
				t.Errorf("errors.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBackendError_Unwrap(t *testing.T) {
	inner := errors.New("inner error")
	wrapped := WrapError("test", "get", "item", inner)

	unwrapped := errors.Unwrap(wrapped)
	if unwrapped != inner {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, inner)
	}
}

func TestBackendError_Error(t *testing.T) {
	tests := []struct {
		name     string
		backend  string
		op       string
		item     string
		err      error
		contains []string
	}{
		{
			name:     "error with item",
			backend:  "bitwarden",
			op:       "get",
			item:     "my-secret",
			err:      ErrNotFound,
			contains: []string{"bitwarden", "get", "my-secret", "not found"},
		},
		{
			name:     "error without item",
			backend:  "pass",
			op:       "init",
			item:     "",
			err:      errors.New("initialization failed"),
			contains: []string{"pass", "init", "initialization failed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WrapError(tt.backend, tt.op, tt.item, tt.err)
			errMsg := err.Error()

			for _, substr := range tt.contains {
				if !contains(errMsg, substr) {
					t.Errorf("Error() = %q, should contain %q", errMsg, substr)
				}
			}
		})
	}
}

func TestWrapError_Nil(t *testing.T) {
	err := WrapError("test", "op", "item", nil)
	if err != nil {
		t.Errorf("WrapError(nil) = %v, want nil", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
