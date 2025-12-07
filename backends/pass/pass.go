// Package pass implements the vaultmux.Backend interface for the pass password manager.
package pass

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/blackwell-systems/vaultmux"
)

func init() {
	vaultmux.RegisterBackend(vaultmux.BackendPass, func(cfg vaultmux.Config) (vaultmux.Backend, error) {
		return New(cfg.StorePath, cfg.Prefix)
	})
}

// Backend implements vaultmux.Backend for pass.
type Backend struct {
	storePath string
	prefix    string
}

// New creates a new pass backend.
func New(storePath, prefix string) (*Backend, error) {
	if storePath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home dir: %w", err)
		}
		storePath = filepath.Join(home, ".password-store")
	}
	if prefix == "" {
		prefix = "dotfiles"
	}
	return &Backend{
		storePath: storePath,
		prefix:    prefix,
	}, nil
}

// Name returns the backend name.
func (b *Backend) Name() string { return "pass" }

// Init checks if pass and gpg are installed and the store exists.
func (b *Backend) Init(ctx context.Context) error {
	// Check pass is installed
	if _, err := exec.LookPath("pass"); err != nil {
		return vaultmux.ErrBackendNotInstalled
	}

	// Check gpg is installed
	if _, err := exec.LookPath("gpg"); err != nil {
		return fmt.Errorf("gpg not installed: %w", vaultmux.ErrBackendNotInstalled)
	}

	// Check store exists
	if _, err := os.Stat(b.storePath); os.IsNotExist(err) {
		return fmt.Errorf("password store not initialized at %s", b.storePath)
	}

	return nil
}

// Close is a no-op for pass.
func (b *Backend) Close() error { return nil }

// IsAuthenticated checks if pass can list items (GPG agent is available).
func (b *Backend) IsAuthenticated(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "pass", "ls")
	return cmd.Run() == nil
}

// Authenticate returns a no-op session since pass uses GPG agent.
func (b *Backend) Authenticate(ctx context.Context) (vaultmux.Session, error) {
	// Verify pass works
	if !b.IsAuthenticated(ctx) {
		return nil, vaultmux.ErrNotAuthenticated
	}
	return &passSession{}, nil
}

// Sync pulls from git if the password store is git-enabled.
func (b *Backend) Sync(ctx context.Context, session vaultmux.Session) error {
	// Check if .git exists in store
	gitDir := filepath.Join(b.storePath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return nil // Not git-enabled, no-op
	}

	// Run: pass git pull
	cmd := exec.CommandContext(ctx, "pass", "git", "pull")
	if err := cmd.Run(); err != nil {
		return vaultmux.WrapError("pass", "sync", "", err)
	}

	return nil
}

// GetItem retrieves a vault item by name.
func (b *Backend) GetItem(ctx context.Context, name string, _ vaultmux.Session) (*vaultmux.Item, error) {
	notes, err := b.GetNotes(ctx, name, nil)
	if err != nil {
		return nil, err
	}
	if notes == "" {
		return nil, vaultmux.ErrNotFound
	}

	return &vaultmux.Item{
		Name:  name,
		Type:  vaultmux.ItemTypeSecureNote,
		Notes: notes,
	}, nil
}

// GetNotes retrieves the content of an item.
func (b *Backend) GetNotes(ctx context.Context, name string, _ vaultmux.Session) (string, error) {
	path := b.itemPath(name)
	cmd := exec.CommandContext(ctx, "pass", "show", path)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "", vaultmux.ErrNotFound
		}
		return "", vaultmux.WrapError("pass", "get", name, err)
	}
	return string(out), nil
}

// ItemExists checks if an item exists in the store.
func (b *Backend) ItemExists(ctx context.Context, name string, _ vaultmux.Session) (bool, error) {
	gpgPath := filepath.Join(b.storePath, b.prefix, name+".gpg")
	_, err := os.Stat(gpgPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// ListItems lists all items in the store under the prefix.
func (b *Backend) ListItems(ctx context.Context, _ vaultmux.Session) ([]*vaultmux.Item, error) {
	prefixPath := filepath.Join(b.storePath, b.prefix)

	var items []*vaultmux.Item
	err := filepath.Walk(prefixPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".gpg") {
			return nil
		}

		// Extract name relative to prefix
		rel, _ := filepath.Rel(prefixPath, path)
		name := strings.TrimSuffix(rel, ".gpg")
		name = filepath.ToSlash(name) // Convert to forward slashes

		items = append(items, &vaultmux.Item{
			Name:     name,
			Type:     vaultmux.ItemTypeSecureNote,
			Modified: info.ModTime(),
		})
		return nil
	})

	if err != nil {
		return nil, vaultmux.WrapError("pass", "list", "", err)
	}

	return items, nil
}

// CreateItem creates a new item.
func (b *Backend) CreateItem(ctx context.Context, name, content string, _ vaultmux.Session) error {
	exists, err := b.ItemExists(ctx, name, nil)
	if err != nil {
		return err
	}
	if exists {
		return vaultmux.ErrAlreadyExists
	}

	path := b.itemPath(name)
	cmd := exec.CommandContext(ctx, "pass", "insert", "-m", path)
	cmd.Stdin = strings.NewReader(content)

	if err := cmd.Run(); err != nil {
		return vaultmux.WrapError("pass", "create", name, err)
	}
	return nil
}

// UpdateItem updates an existing item.
func (b *Backend) UpdateItem(ctx context.Context, name, content string, _ vaultmux.Session) error {
	exists, err := b.ItemExists(ctx, name, nil)
	if err != nil {
		return err
	}
	if !exists {
		return vaultmux.ErrNotFound
	}

	path := b.itemPath(name)
	cmd := exec.CommandContext(ctx, "pass", "insert", "-m", "-f", path)
	cmd.Stdin = strings.NewReader(content)

	if err := cmd.Run(); err != nil {
		return vaultmux.WrapError("pass", "update", name, err)
	}
	return nil
}

// DeleteItem removes an item.
func (b *Backend) DeleteItem(ctx context.Context, name string, _ vaultmux.Session) error {
	path := b.itemPath(name)
	cmd := exec.CommandContext(ctx, "pass", "rm", "-f", path)
	if err := cmd.Run(); err != nil {
		return vaultmux.WrapError("pass", "delete", name, err)
	}
	return nil
}

// ListLocations lists top-level directories as "locations".
func (b *Backend) ListLocations(ctx context.Context, _ vaultmux.Session) ([]string, error) {
	prefixPath := filepath.Join(b.storePath, b.prefix)

	entries, err := os.ReadDir(prefixPath)
	if err != nil {
		return nil, vaultmux.WrapError("pass", "list-locations", "", err)
	}

	var locations []string
	for _, entry := range entries {
		if entry.IsDir() {
			locations = append(locations, entry.Name())
		}
	}

	return locations, nil
}

// LocationExists checks if a location (directory) exists.
func (b *Backend) LocationExists(ctx context.Context, name string, _ vaultmux.Session) (bool, error) {
	path := filepath.Join(b.storePath, b.prefix, name)
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

// CreateLocation creates a new location (directory).
func (b *Backend) CreateLocation(ctx context.Context, name string, _ vaultmux.Session) error {
	path := filepath.Join(b.storePath, b.prefix, name)
	if err := os.MkdirAll(path, 0755); err != nil {
		return vaultmux.WrapError("pass", "create-location", name, err)
	}
	return nil
}

// ListItemsInLocation lists items within a specific location.
func (b *Backend) ListItemsInLocation(ctx context.Context, locType, locValue string, _ vaultmux.Session) ([]*vaultmux.Item, error) {
	// For pass, locType is ignored (always directory-based)
	locationPath := filepath.Join(b.storePath, b.prefix, locValue)

	var items []*vaultmux.Item
	err := filepath.Walk(locationPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".gpg") {
			return nil
		}

		rel, _ := filepath.Rel(locationPath, path)
		name := strings.TrimSuffix(rel, ".gpg")
		name = filepath.ToSlash(name)

		items = append(items, &vaultmux.Item{
			Name:     name,
			Type:     vaultmux.ItemTypeSecureNote,
			Location: locValue,
			Modified: info.ModTime(),
		})
		return nil
	})

	if err != nil {
		return nil, vaultmux.WrapError("pass", "list-items-in-location", locValue, err)
	}

	return items, nil
}

// itemPath returns the full path for an item.
func (b *Backend) itemPath(name string) string {
	return filepath.Join(b.prefix, name)
}

// passSession implements vaultmux.Session for pass (no-op).
type passSession struct{}

func (s *passSession) Token() string                     { return "" }
func (s *passSession) IsValid(ctx context.Context) bool  { return true }
func (s *passSession) Refresh(ctx context.Context) error { return nil }
func (s *passSession) ExpiresAt() time.Time              { return time.Time{} }
