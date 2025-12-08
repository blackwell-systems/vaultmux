package gcpsecrets

import (
	"context"
	"time"
)

// gcpSession implements vaultmux.Session for GCP Secret Manager.
// Similar to AWS, GCP uses service account credentials that are either:
//   - Long-lived (service account JSON files)
//   - Short-lived (GKE workload identity, auto-refreshed by SDK)
//
// The SDK handles token refresh automatically via Application Default Credentials (ADC).
type gcpSession struct {
	projectID string  // GCP project ID is required for all operations
	backend   *Backend
}

// Token returns the GCP project ID (not a traditional token).
// GCP credentials are managed by the SDK internally via ADC.
func (s *gcpSession) Token() string {
	return s.projectID
}

// IsValid checks if the project ID is set and credentials are available.
// GCP SDK validates credentials on first API call, so this is a lightweight check.
func (s *gcpSession) IsValid(ctx context.Context) bool {
	// Project ID is required
	if s.projectID == "" {
		return false
	}

	// Backend client must be initialized
	if s.backend == nil || s.backend.client == nil {
		return false
	}

	return true
}

// Refresh re-initializes the GCP client to pick up new credentials.
// Typically not needed as the SDK auto-refreshes credentials.
func (s *gcpSession) Refresh(ctx context.Context) error {
	return s.backend.initGCPClient(ctx)
}

// ExpiresAt returns zero time because GCP credentials don't have simple expiration:
//   - Service account keys are long-lived (rotated manually)
//   - GKE workload identity tokens are auto-refreshed by the SDK
//   - Compute Engine service accounts are auto-refreshed by the SDK
func (s *gcpSession) ExpiresAt() time.Time {
	return time.Time{} // Zero value indicates no explicit expiration
}
