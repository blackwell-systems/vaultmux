package vaultmux

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SessionCache handles session persistence to disk.
type SessionCache struct {
	path string
	ttl  time.Duration
}

// CachedSession represents a persisted session.
type CachedSession struct {
	Token   string    `json:"token"`
	Created time.Time `json:"created"`
	Expires time.Time `json:"expires"`
	Backend string    `json:"backend"`
}

// NewSessionCache creates a session cache.
func NewSessionCache(path string, ttl time.Duration) *SessionCache {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	os.MkdirAll(dir, 0755)

	return &SessionCache{
		path: path,
		ttl:  ttl,
	}
}

// Load reads a cached session from disk.
func (c *SessionCache) Load() (*CachedSession, error) {
	data, err := os.ReadFile(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No cached session
		}
		return nil, fmt.Errorf("read session cache: %w", err)
	}

	var session CachedSession
	if err := json.Unmarshal(data, &session); err != nil {
		// Invalid cache - remove it
		os.Remove(c.path)
		return nil, nil
	}

	// Check if expired
	if time.Now().After(session.Expires) {
		os.Remove(c.path)
		return nil, nil
	}

	return &session, nil
}

// Save writes a session to disk.
func (c *SessionCache) Save(token, backend string) error {
	now := time.Now()
	session := CachedSession{
		Token:   token,
		Created: now,
		Expires: now.Add(c.ttl),
		Backend: backend,
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	// Write with restricted permissions
	if err := os.WriteFile(c.path, data, 0600); err != nil {
		return fmt.Errorf("write session cache: %w", err)
	}

	return nil
}

// Clear removes the cached session.
func (c *SessionCache) Clear() error {
	err := os.Remove(c.path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// AutoRefreshSession wraps a session with automatic refresh capability.
type AutoRefreshSession struct {
	inner   Session
	backend Backend
}

// NewAutoRefreshSession creates a session that auto-refreshes when expired.
func NewAutoRefreshSession(session Session, backend Backend) Session {
	return &AutoRefreshSession{
		inner:   session,
		backend: backend,
	}
}

// Token returns the session token, refreshing if needed.
func (s *AutoRefreshSession) Token() string {
	ctx := context.Background()
	if !s.inner.IsValid(ctx) {
		// Attempt refresh
		if err := s.inner.Refresh(ctx); err != nil {
			// Refresh failed - would need to re-authenticate
			// For now, return expired token (operations will fail)
			return s.inner.Token()
		}
	}
	return s.inner.Token()
}

// IsValid checks if the inner session is valid.
func (s *AutoRefreshSession) IsValid(ctx context.Context) bool {
	return s.inner.IsValid(ctx)
}

// Refresh delegates to the inner session.
func (s *AutoRefreshSession) Refresh(ctx context.Context) error {
	return s.inner.Refresh(ctx)
}

// ExpiresAt returns when the session expires.
func (s *AutoRefreshSession) ExpiresAt() time.Time {
	return s.inner.ExpiresAt()
}
