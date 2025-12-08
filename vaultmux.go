// Package vaultmux provides a unified interface for interacting with multiple
// secret management backends including Bitwarden, 1Password, pass (Unix password
// manager), Windows Credential Manager, AWS Secrets Manager, and Google Cloud
// Secret Manager.
//
// The library supports three integration patterns:
//   - CLI backends: Bitwarden (bw), 1Password (op), pass
//   - OS-native: Windows Credential Manager (PowerShell)
//   - SDK-based: AWS Secrets Manager (aws-sdk-go-v2), Google Cloud Secret Manager (cloud.google.com/go)
//
// Basic usage:
//
//	import (
//	    "github.com/blackwell-systems/vaultmux"
//	    _ "github.com/blackwell-systems/vaultmux/backends/pass"
//	    _ "github.com/blackwell-systems/vaultmux/backends/awssecrets"
//	    _ "github.com/blackwell-systems/vaultmux/backends/gcpsecrets"
//	)
//
//	// Using pass backend
//	backend, err := vaultmux.New(vaultmux.Config{
//	    Backend: vaultmux.BackendPass,
//	    Prefix:  "myapp",
//	})
//
//	// Using AWS Secrets Manager
//	backend, err := vaultmux.New(vaultmux.Config{
//	    Backend: vaultmux.BackendAWSSecretsManager,
//	    Options: map[string]string{
//	        "region": "us-west-2",
//	        "prefix": "myapp/",
//	    },
//	})
//
//	// Using Google Cloud Secret Manager
//	backend, err := vaultmux.New(vaultmux.Config{
//	    Backend: vaultmux.BackendGCPSecretManager,
//	    Options: map[string]string{
//	        "project_id": "my-gcp-project",
//	        "prefix":     "myapp-",
//	    },
//	})
//
// See the package documentation for more examples.
package vaultmux // import "github.com/blackwell-systems/vaultmux"

import (
	"context"
	"errors"
	"time"
)

// Backend represents a secret storage backend.
// Implementations: Bitwarden, 1Password, pass, Windows Credential Manager, AWS Secrets Manager, Google Cloud Secret Manager
type Backend interface {
	// Metadata
	Name() string

	// Lifecycle
	Init(ctx context.Context) error
	Close() error

	// Authentication
	IsAuthenticated(ctx context.Context) bool
	Authenticate(ctx context.Context) (Session, error)

	// Sync pulls latest from server (no-op for pass)
	Sync(ctx context.Context, session Session) error

	// Item operations (CRUD)
	GetItem(ctx context.Context, name string, session Session) (*Item, error)
	GetNotes(ctx context.Context, name string, session Session) (string, error)
	ItemExists(ctx context.Context, name string, session Session) (bool, error)
	ListItems(ctx context.Context, session Session) ([]*Item, error)

	// Mutations
	CreateItem(ctx context.Context, name, content string, session Session) error
	UpdateItem(ctx context.Context, name, content string, session Session) error
	DeleteItem(ctx context.Context, name string, session Session) error

	// Location management (folders/vaults)
	LocationManager
}

// Session represents an authenticated session.
// Opaque to callers - backend-specific internals.
type Session interface {
	// Token returns the session token (empty for pass).
	Token() string

	// IsValid checks if the session is still valid.
	IsValid(ctx context.Context) bool

	// Refresh attempts to refresh an expired session.
	Refresh(ctx context.Context) error

	// ExpiresAt returns when the session expires (zero for non-expiring).
	ExpiresAt() time.Time
}

// LocationManager handles organizational units (folders, vaults, etc.)
type LocationManager interface {
	ListLocations(ctx context.Context, session Session) ([]string, error)
	LocationExists(ctx context.Context, name string, session Session) (bool, error)
	CreateLocation(ctx context.Context, name string, session Session) error
	ListItemsInLocation(ctx context.Context, locType, locValue string, session Session) ([]*Item, error)
}

// Item represents a vault item.
type Item struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Type     ItemType          `json:"type"`
	Notes    string            `json:"notes,omitempty"`
	Fields   map[string]string `json:"fields,omitempty"`
	Location string            `json:"location,omitempty"` // Folder/vault
	Created  time.Time         `json:"created,omitempty"`
	Modified time.Time         `json:"modified,omitempty"`
}

// ItemType indicates the type of vault item.
type ItemType int

const (
	// ItemTypeSecureNote represents a secure note item.
	ItemTypeSecureNote ItemType = iota
	// ItemTypeLogin represents a login credential item.
	ItemTypeLogin
	// ItemTypeSSHKey represents an SSH key item.
	ItemTypeSSHKey
	// ItemTypeIdentity represents an identity item.
	ItemTypeIdentity
	// ItemTypeCard represents a credit card item.
	ItemTypeCard
)

// String returns the string representation of ItemType.
func (t ItemType) String() string {
	switch t {
	case ItemTypeSecureNote:
		return "SecureNote"
	case ItemTypeLogin:
		return "Login"
	case ItemTypeSSHKey:
		return "SSHKey"
	case ItemTypeIdentity:
		return "Identity"
	case ItemTypeCard:
		return "Card"
	default:
		return "Unknown"
	}
}

// Common errors
var (
	// ErrNotFound indicates the item doesn't exist.
	ErrNotFound = errors.New("item not found")

	// ErrAlreadyExists indicates the item already exists.
	ErrAlreadyExists = errors.New("item already exists")

	// ErrNotAuthenticated indicates no valid session.
	ErrNotAuthenticated = errors.New("not authenticated")

	// ErrSessionExpired indicates the session has expired.
	ErrSessionExpired = errors.New("session expired")

	// ErrBackendNotInstalled indicates the CLI tool is missing.
	ErrBackendNotInstalled = errors.New("backend CLI not installed")

	// ErrBackendLocked indicates the vault is locked.
	ErrBackendLocked = errors.New("vault is locked")

	// ErrPermissionDenied indicates insufficient permissions.
	ErrPermissionDenied = errors.New("permission denied")
)
