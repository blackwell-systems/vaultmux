//go:build windows

package wincred

import (
	"context"
	"testing"
)

func TestNew_Windows(t *testing.T) {
	tests := []struct {
		name       string
		prefix     string
		wantPrefix string
	}{
		{
			name:       "with custom prefix",
			prefix:     "myapp",
			wantPrefix: "myapp",
		},
		{
			name:       "with empty prefix",
			prefix:     "",
			wantPrefix: "vaultmux",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, err := New(tt.prefix)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			if backend.prefix != tt.wantPrefix {
				t.Errorf("prefix = %q, want %q", backend.prefix, tt.wantPrefix)
			}
		})
	}
}

func TestBackend_Name_Windows(t *testing.T) {
	backend, _ := New("test")
	if name := backend.Name(); name != "wincred" {
		t.Errorf("Name() = %q, want %q", name, "wincred")
	}
}

func TestBackend_IsAuthenticated_Windows(t *testing.T) {
	backend, _ := New("test")
	ctx := context.Background()

	// Should always return true (OS handles auth)
	if !backend.IsAuthenticated(ctx) {
		t.Error("IsAuthenticated() = false, want true")
	}
}

func TestBackend_Authenticate_Windows(t *testing.T) {
	backend, _ := New("test")
	ctx := context.Background()

	session, err := backend.Authenticate(ctx)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if session == nil {
		t.Error("Authenticate() returned nil session")
	}
}

func TestBackend_credentialTarget(t *testing.T) {
	tests := []struct {
		name       string
		prefix     string
		itemName   string
		wantTarget string
	}{
		{
			name:       "basic target",
			prefix:     "vaultmux",
			itemName:   "test-key",
			wantTarget: "vaultmux:test-key",
		},
		{
			name:       "custom prefix",
			prefix:     "myapp",
			itemName:   "api-token",
			wantTarget: "myapp:api-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, _ := New(tt.prefix)
			target := backend.credentialTarget(tt.itemName)
			if target != tt.wantTarget {
				t.Errorf("credentialTarget() = %q, want %q", target, tt.wantTarget)
			}
		})
	}
}

func TestEscapePowerShellString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no quotes",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "single quote",
			input: "it's",
			want:  "it''s",
		},
		{
			name:  "multiple quotes",
			input: "'hello' 'world'",
			want:  "''hello'' ''world''",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapePowerShellString(tt.input)
			if got != tt.want {
				t.Errorf("escapePowerShellString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBackend_LocationManagement_Windows(t *testing.T) {
	backend, _ := New("test")
	ctx := context.Background()

	t.Run("ListLocations", func(t *testing.T) {
		locs, err := backend.ListLocations(ctx, nil)
		if err != nil {
			t.Fatalf("ListLocations() error = %v", err)
		}
		if len(locs) != 0 {
			t.Errorf("ListLocations() returned %d locations, want 0", len(locs))
		}
	})

	t.Run("LocationExists", func(t *testing.T) {
		exists, err := backend.LocationExists(ctx, "test", nil)
		if err != nil {
			t.Fatalf("LocationExists() error = %v", err)
		}
		if exists {
			t.Error("LocationExists() = true, want false")
		}
	})

	t.Run("CreateLocation", func(t *testing.T) {
		if err := backend.CreateLocation(ctx, "test", nil); err != nil {
			t.Errorf("CreateLocation() error = %v, want nil", err)
		}
	})

	t.Run("ListItemsInLocation", func(t *testing.T) {
		items, err := backend.ListItemsInLocation(ctx, "folder", "test", nil)
		if err != nil {
			t.Fatalf("ListItemsInLocation() error = %v", err)
		}
		if len(items) != 0 {
			t.Errorf("ListItemsInLocation() returned %d items, want 0", len(items))
		}
	})
}
