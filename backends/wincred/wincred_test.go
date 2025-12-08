package wincred

import (
	"context"
	"runtime"
	"testing"

	"github.com/blackwell-systems/vaultmux"
)

func TestNew_Unix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix-specific test")
	}

	_, err := New("test")
	if err == nil {
		t.Error("New() should return error on non-Windows")
	}
}

func TestBackend_UnixStub(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix-specific test")
	}

	// Test that factory registration returns error
	backend, err := vaultmux.New(vaultmux.Config{
		Backend: vaultmux.BackendWindowsCredentialManager,
	})
	if err == nil {
		t.Error("New() should return error for wincred on non-Windows")
	}
	if backend != nil {
		t.Error("New() should return nil backend on error")
	}
}

func TestSession(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test")
	}

	session := &winCredSession{}

	t.Run("Token", func(t *testing.T) {
		if token := session.Token(); token != "" {
			t.Errorf("Token() = %q, want empty string", token)
		}
	})

	t.Run("IsValid", func(t *testing.T) {
		ctx := context.Background()
		if !session.IsValid(ctx) {
			t.Error("IsValid() = false, want true")
		}
	})

	t.Run("Refresh", func(t *testing.T) {
		ctx := context.Background()
		if err := session.Refresh(ctx); err != nil {
			t.Errorf("Refresh() error = %v, want nil", err)
		}
	})

	t.Run("ExpiresAt", func(t *testing.T) {
		expires := session.ExpiresAt()
		if !expires.IsZero() {
			t.Errorf("ExpiresAt() = %v, want zero time", expires)
		}
	})
}
