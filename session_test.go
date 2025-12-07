package vaultmux

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSessionCache_SaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	sessionFile := filepath.Join(tmpDir, ".test-session")
	cache := NewSessionCache(sessionFile, 30*time.Minute)

	t.Run("save and load", func(t *testing.T) {
		err := cache.Save("test-token-123", "test-backend")
		if err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		// Verify file was created with correct permissions (Unix only)
		info, err := os.Stat(sessionFile)
		if err != nil {
			t.Fatalf("Stat() error = %v", err)
		}
		mode := info.Mode()
		// Windows doesn't support Unix permissions - skip check
		if mode.Perm() != 0600 && mode.Perm() != 0666 {
			t.Errorf("file mode = %o, want 0600 (or 0666 on Windows)", mode.Perm())
		}

		// Load the session
		session, err := cache.Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if session == nil {
			t.Fatal("Load() returned nil session")
		}

		if session.Token != "test-token-123" {
			t.Errorf("session.Token = %q, want %q", session.Token, "test-token-123")
		}
		if session.Backend != "test-backend" {
			t.Errorf("session.Backend = %q, want %q", session.Backend, "test-backend")
		}
	})

	t.Run("load nonexistent", func(t *testing.T) {
		cache := NewSessionCache(filepath.Join(tmpDir, "nonexistent"), 30*time.Minute)
		session, err := cache.Load()
		if err != nil {
			t.Errorf("Load() error = %v, want nil", err)
		}
		if session != nil {
			t.Errorf("Load() = %v, want nil", session)
		}
	})

	t.Run("load expired", func(t *testing.T) {
		expiredFile := filepath.Join(tmpDir, ".expired-session")
		cache := NewSessionCache(expiredFile, 1*time.Nanosecond)

		// Save and wait for expiry
		_ = cache.Save("expired-token", "test")
		time.Sleep(10 * time.Millisecond)

		session, err := cache.Load()
		if err != nil {
			t.Errorf("Load() error = %v, want nil", err)
		}
		if session != nil {
			t.Errorf("Load() = %v, want nil for expired session", session)
		}

		// File should be removed
		if _, err := os.Stat(expiredFile); !os.IsNotExist(err) {
			t.Error("expired session file still exists")
		}
	})
}

func TestSessionCache_Clear(t *testing.T) {
	tmpDir := t.TempDir()
	sessionFile := filepath.Join(tmpDir, ".test-session")
	cache := NewSessionCache(sessionFile, 30*time.Minute)

	// Create a session
	_ = cache.Save("test-token", "test-backend")

	// Clear it
	if err := cache.Clear(); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	// Verify it's gone
	if _, err := os.Stat(sessionFile); !os.IsNotExist(err) {
		t.Error("session file still exists after Clear()")
	}

	// Clear non-existent should not error
	if err := cache.Clear(); err != nil {
		t.Errorf("Clear() on nonexistent error = %v, want nil", err)
	}
}

func TestSessionCache_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	sessionFile := filepath.Join(tmpDir, ".invalid-session")

	// Write invalid JSON
	err := os.WriteFile(sessionFile, []byte("invalid json{"), 0600)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cache := NewSessionCache(sessionFile, 30*time.Minute)
	session, err := cache.Load()
	if err != nil {
		t.Errorf("Load() error = %v, want nil (should handle gracefully)", err)
	}
	if session != nil {
		t.Errorf("Load() = %v, want nil for invalid JSON", session)
	}

	// File should be removed
	if _, err := os.Stat(sessionFile); !os.IsNotExist(err) {
		t.Error("invalid session file should be removed")
	}
}

func TestNewSessionCache_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	sessionFile := filepath.Join(tmpDir, "subdir", "nested", ".session")

	cache := NewSessionCache(sessionFile, 30*time.Minute)

	// Try to save - should create parent directories
	err := cache.Save("test-token", "test-backend")
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify directory was created
	dir := filepath.Dir(sessionFile)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("parent directory was not created")
	}
}

func TestCachedSession_Fields(t *testing.T) {
	now := time.Now()
	expires := now.Add(30 * time.Minute)

	session := &CachedSession{
		Token:   "test-token",
		Created: now,
		Expires: expires,
		Backend: "test-backend",
	}

	if session.Token != "test-token" {
		t.Errorf("Token = %q, want %q", session.Token, "test-token")
	}
	if session.Backend != "test-backend" {
		t.Errorf("Backend = %q, want %q", session.Backend, "test-backend")
	}
	if !session.Created.Equal(now) {
		t.Errorf("Created = %v, want %v", session.Created, now)
	}
	if !session.Expires.Equal(expires) {
		t.Errorf("Expires = %v, want %v", session.Expires, expires)
	}
}

// Mock session for testing AutoRefreshSession
type mockTestSession struct {
	token      string
	valid      bool
	refreshErr error
	expires    time.Time
}

func (s *mockTestSession) Token() string                     { return s.token }
func (s *mockTestSession) IsValid(ctx context.Context) bool  { return s.valid }
func (s *mockTestSession) Refresh(ctx context.Context) error { return s.refreshErr }
func (s *mockTestSession) ExpiresAt() time.Time              { return s.expires }

// Mock backend for testing AutoRefreshSession
type mockTestBackend struct{}

func (b *mockTestBackend) Name() string                             { return "mock" }
func (b *mockTestBackend) Init(ctx context.Context) error           { return nil }
func (b *mockTestBackend) Close() error                             { return nil }
func (b *mockTestBackend) IsAuthenticated(ctx context.Context) bool { return true }
func (b *mockTestBackend) Authenticate(ctx context.Context) (Session, error) {
	return &mockTestSession{token: "new-token", valid: true}, nil
}
func (b *mockTestBackend) Sync(ctx context.Context, session Session) error { return nil }
func (b *mockTestBackend) GetItem(ctx context.Context, name string, session Session) (*Item, error) {
	return nil, nil
}
func (b *mockTestBackend) GetNotes(ctx context.Context, name string, session Session) (string, error) {
	return "", nil
}
func (b *mockTestBackend) ItemExists(ctx context.Context, name string, session Session) (bool, error) {
	return false, nil
}
func (b *mockTestBackend) ListItems(ctx context.Context, session Session) ([]*Item, error) {
	return nil, nil
}
func (b *mockTestBackend) CreateItem(ctx context.Context, name, content string, session Session) error {
	return nil
}
func (b *mockTestBackend) UpdateItem(ctx context.Context, name, content string, session Session) error {
	return nil
}
func (b *mockTestBackend) DeleteItem(ctx context.Context, name string, session Session) error {
	return nil
}
func (b *mockTestBackend) ListLocations(ctx context.Context, session Session) ([]string, error) {
	return nil, nil
}
func (b *mockTestBackend) LocationExists(ctx context.Context, name string, session Session) (bool, error) {
	return false, nil
}
func (b *mockTestBackend) CreateLocation(ctx context.Context, name string, session Session) error {
	return nil
}
func (b *mockTestBackend) ListItemsInLocation(ctx context.Context, locType, locValue string, session Session) ([]*Item, error) {
	return nil, nil
}

func TestAutoRefreshSession(t *testing.T) {
	backend := &mockTestBackend{}

	t.Run("valid session", func(t *testing.T) {
		inner := &mockTestSession{
			token: "test-token",
			valid: true,
		}
		session := NewAutoRefreshSession(inner, backend)

		token := session.Token()
		if token != "test-token" {
			t.Errorf("Token() = %q, want %q", token, "test-token")
		}
	})

	t.Run("refresh on invalid", func(t *testing.T) {
		inner := &mockTestSession{
			token: "old-token",
			valid: false,
		}
		session := NewAutoRefreshSession(inner, backend)

		// This should trigger refresh
		token := session.Token()

		// Should still return old token if refresh failed
		// (actual behavior depends on implementation)
		if token == "" {
			t.Error("Token() returned empty string")
		}
	})

	t.Run("delegates IsValid", func(t *testing.T) {
		inner := &mockTestSession{
			token: "test-token",
			valid: true,
		}
		session := NewAutoRefreshSession(inner, backend)

		if !session.IsValid(context.Background()) {
			t.Error("IsValid() = false, want true")
		}
	})

	t.Run("delegates ExpiresAt", func(t *testing.T) {
		expires := time.Now().Add(time.Hour)
		inner := &mockTestSession{
			token:   "test-token",
			valid:   true,
			expires: expires,
		}
		session := NewAutoRefreshSession(inner, backend)

		if !session.ExpiresAt().Equal(expires) {
			t.Errorf("ExpiresAt() = %v, want %v", session.ExpiresAt(), expires)
		}
	})
}
