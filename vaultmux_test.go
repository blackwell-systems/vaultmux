package vaultmux

import (
	"context"
	"errors"
	"testing"
)

func TestBackendType(t *testing.T) {
	tests := []struct {
		name    string
		backend BackendType
		valid   bool
	}{
		{"bitwarden", BackendBitwarden, true},
		{"1password", BackendOnePassword, true},
		{"pass", BackendPass, true},
		{"invalid", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the constant exists
			if tt.valid {
				if tt.backend == "" {
					t.Errorf("BackendType %q should not be empty", tt.name)
				}
			}
		})
	}
}

func TestItemType_String(t *testing.T) {
	tests := []struct {
		itemType ItemType
		want     string
	}{
		{ItemTypeSecureNote, "SecureNote"},
		{ItemTypeLogin, "Login"},
		{ItemTypeSSHKey, "SSHKey"},
		{ItemTypeIdentity, "Identity"},
		{ItemTypeCard, "Card"},
		{ItemType(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.itemType.String(); got != tt.want {
				t.Errorf("ItemType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommonErrors(t *testing.T) {
	// Verify all error constants are defined
	if ErrNotFound == nil {
		t.Error("ErrNotFound should be defined")
	}
	if ErrAlreadyExists == nil {
		t.Error("ErrAlreadyExists should be defined")
	}
	if ErrNotAuthenticated == nil {
		t.Error("ErrNotAuthenticated should be defined")
	}
	if ErrSessionExpired == nil {
		t.Error("ErrSessionExpired should be defined")
	}
	if ErrBackendNotInstalled == nil {
		t.Error("ErrBackendNotInstalled should be defined")
	}
	if ErrBackendLocked == nil {
		t.Error("ErrBackendLocked should be defined")
	}
	if ErrPermissionDenied == nil {
		t.Error("ErrPermissionDenied should be defined")
	}
}

func TestBackendError(t *testing.T) {
	baseErr := errors.New("base error")
	wrappedErr := WrapError("test-backend", "get", "item-name", baseErr)

	// Test error message format
	expected := "test-backend: get \"item-name\": base error"
	if wrappedErr.Error() != expected {
		t.Errorf("WrapError().Error() = %q, want %q", wrappedErr.Error(), expected)
	}

	// Test unwrapping
	if !errors.Is(wrappedErr, baseErr) {
		t.Error("WrapError should wrap the original error")
	}

	// Test without item name
	wrappedErr2 := WrapError("backend", "sync", "", baseErr)
	expected2 := "backend: sync: base error"
	if wrappedErr2.Error() != expected2 {
		t.Errorf("WrapError().Error() = %q, want %q", wrappedErr2.Error(), expected2)
	}

	// Test nil error
	if err := WrapError("backend", "op", "item", nil); err != nil {
		t.Error("WrapError(nil) should return nil")
	}
}

func TestConfig(t *testing.T) {
	cfg := Config{
		Backend:     BackendPass,
		StorePath:   "/tmp/store",
		Prefix:      "test",
		SessionFile: "/tmp/session",
		SessionTTL:  1800,
		Options:     map[string]string{"key": "value"},
	}

	if cfg.Backend != BackendPass {
		t.Errorf("Config.Backend = %v, want %v", cfg.Backend, BackendPass)
	}

	if cfg.SessionTTL != 1800 {
		t.Errorf("Config.SessionTTL = %v, want 1800", cfg.SessionTTL)
	}
}

func TestItem(t *testing.T) {
	item := Item{
		ID:       "test-id",
		Name:     "test-item",
		Type:     ItemTypeSecureNote,
		Notes:    "secret content",
		Location: "folder1",
	}

	if item.ID != "test-id" {
		t.Errorf("Item.ID = %v, want 'test-id'", item.ID)
	}

	if item.Type != ItemTypeSecureNote {
		t.Errorf("Item.Type = %v, want ItemTypeSecureNote", item.Type)
	}
}

// Example test showing how to use the mock backend
func TestExampleWithMock(t *testing.T) {
	// This would import the mock package in real tests
	// For now, this is just a demonstration of the pattern
	ctx := context.Background()

	// In a real test:
	// backend := mock.New()
	// backend.SetItem("test", "value")
	//
	// session, err := backend.Authenticate(ctx)
	// if err != nil {
	//     t.Fatal(err)
	// }
	//
	// notes, err := backend.GetNotes(ctx, "test", session)
	// if err != nil {
	//     t.Fatal(err)
	// }
	// if notes != "value" {
	//     t.Errorf("got %q, want %q", notes, "value")
	// }

	_ = ctx // Prevent unused warning
}
