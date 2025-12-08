package wincred

import (
	"context"
	"time"
)

// winCredSession represents a Windows Credential Manager session.
// Since Windows handles authentication at the OS level, this is a no-op session.
type winCredSession struct{}

// Token returns an empty string since Windows Credential Manager doesn't use tokens.
func (s *winCredSession) Token() string {
	return ""
}

// IsValid always returns true since OS handles authentication.
func (s *winCredSession) IsValid(ctx context.Context) bool {
	return true
}

// Refresh is a no-op since there's no session to refresh.
func (s *winCredSession) Refresh(ctx context.Context) error {
	return nil
}

// ExpiresAt returns zero time since the session never expires.
func (s *winCredSession) ExpiresAt() time.Time {
	return time.Time{}
}
