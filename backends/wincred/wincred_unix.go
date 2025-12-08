//go:build !windows

// Package wincred provides a stub implementation for non-Windows platforms.
package wincred

import (
	"context"
	"errors"

	"github.com/blackwell-systems/vaultmux"
)

func init() {
	vaultmux.RegisterBackend(vaultmux.BackendWindowsCredentialManager, func(cfg vaultmux.Config) (vaultmux.Backend, error) {
		return nil, errors.New("Windows Credential Manager is only available on Windows")
	})
}

// Backend is a stub for non-Windows platforms.
type Backend struct{}

// New returns an error on non-Windows platforms.
func New(prefix string) (*Backend, error) {
	return nil, errors.New("Windows Credential Manager is only available on Windows")
}

// Name returns the backend name.
func (b *Backend) Name() string { return "wincred" }

// Init returns an error.
func (b *Backend) Init(ctx context.Context) error {
	return errors.New("Windows Credential Manager is only available on Windows")
}

// Close is a no-op.
func (b *Backend) Close() error { return nil }

// IsAuthenticated returns false.
func (b *Backend) IsAuthenticated(ctx context.Context) bool { return false }

// Authenticate returns an error.
func (b *Backend) Authenticate(ctx context.Context) (vaultmux.Session, error) {
	return nil, errors.New("Windows Credential Manager is only available on Windows")
}

// Sync returns an error.
func (b *Backend) Sync(ctx context.Context, session vaultmux.Session) error {
	return errors.New("Windows Credential Manager is only available on Windows")
}

// GetItem returns an error.
func (b *Backend) GetItem(ctx context.Context, name string, session vaultmux.Session) (*vaultmux.Item, error) {
	return nil, errors.New("Windows Credential Manager is only available on Windows")
}

// GetNotes returns an error.
func (b *Backend) GetNotes(ctx context.Context, name string, session vaultmux.Session) (string, error) {
	return "", errors.New("Windows Credential Manager is only available on Windows")
}

// ItemExists returns an error.
func (b *Backend) ItemExists(ctx context.Context, name string, session vaultmux.Session) (bool, error) {
	return false, errors.New("Windows Credential Manager is only available on Windows")
}

// ListItems returns an error.
func (b *Backend) ListItems(ctx context.Context, session vaultmux.Session) ([]*vaultmux.Item, error) {
	return nil, errors.New("Windows Credential Manager is only available on Windows")
}

// CreateItem returns an error.
func (b *Backend) CreateItem(ctx context.Context, name, content string, session vaultmux.Session) error {
	return errors.New("Windows Credential Manager is only available on Windows")
}

// UpdateItem returns an error.
func (b *Backend) UpdateItem(ctx context.Context, name, content string, session vaultmux.Session) error {
	return errors.New("Windows Credential Manager is only available on Windows")
}

// DeleteItem returns an error.
func (b *Backend) DeleteItem(ctx context.Context, name string, session vaultmux.Session) error {
	return errors.New("Windows Credential Manager is only available on Windows")
}

// ListLocations returns an error.
func (b *Backend) ListLocations(ctx context.Context, session vaultmux.Session) ([]string, error) {
	return nil, errors.New("Windows Credential Manager is only available on Windows")
}

// LocationExists returns an error.
func (b *Backend) LocationExists(ctx context.Context, name string, session vaultmux.Session) (bool, error) {
	return false, errors.New("Windows Credential Manager is only available on Windows")
}

// CreateLocation returns an error.
func (b *Backend) CreateLocation(ctx context.Context, name string, session vaultmux.Session) error {
	return errors.New("Windows Credential Manager is only available on Windows")
}

// ListItemsInLocation returns an error.
func (b *Backend) ListItemsInLocation(ctx context.Context, locType, locValue string, session vaultmux.Session) ([]*vaultmux.Item, error) {
	return nil, errors.New("Windows Credential Manager is only available on Windows")
}
