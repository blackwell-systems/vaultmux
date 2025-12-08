package azurekeyvault

import (
	"context"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
)

// azureSession implements vaultmux.Session for Azure Key Vault.
// Unlike CLI-based backends, Azure sessions wrap Azure AD credentials
// which are long-lived or managed by the SDK.
type azureSession struct {
	// Azure AD credential (service principal, managed identity, CLI, etc.)
	credential azcore.TokenCredential

	// Vault URL for this session (required for all operations)
	vaultURL string

	// Reference to backend for operations
	backend *Backend
}

// Token returns the vault URL as the session identifier.
// Azure doesn't have a simple "session token" - authentication is handled
// via Azure AD tokens that the SDK manages internally.
func (s *azureSession) Token() string {
	return s.vaultURL
}

// IsValid checks if the session is still valid.
// For Azure, this means the credential and vault URL are available.
func (s *azureSession) IsValid(ctx context.Context) bool {
	if s.vaultURL == "" {
		return false
	}
	if s.credential == nil {
		return false
	}
	if s.backend == nil || s.backend.client == nil {
		return false
	}
	return true
}

// Refresh is a no-op for Azure Key Vault.
// Azure AD credentials are automatically refreshed by the SDK when needed.
func (s *azureSession) Refresh(ctx context.Context) error {
	return nil
}

// ExpiresAt returns zero time because Azure AD credentials don't have
// a simple expiration that we manage - the SDK handles token refresh.
func (s *azureSession) ExpiresAt() time.Time {
	return time.Time{} // No expiration (SDK manages token lifecycle)
}
