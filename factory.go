package vaultmux

import (
	"fmt"
	"sync"
)

// BackendType identifies a vault backend.
type BackendType string

const (
	// BackendBitwarden represents the Bitwarden backend.
	BackendBitwarden BackendType = "bitwarden"
	// BackendOnePassword represents the 1Password backend.
	BackendOnePassword BackendType = "1password"
	// BackendPass represents the pass (Unix password manager) backend.
	BackendPass BackendType = "pass"
	// BackendWindowsCredentialManager represents the Windows Credential Manager backend.
	BackendWindowsCredentialManager BackendType = "wincred"
	// BackendAWSSecretsManager represents the AWS Secrets Manager backend.
	BackendAWSSecretsManager BackendType = "awssecrets"
)

// Config holds vault configuration.
type Config struct {
	// Backend type: "bitwarden", "1password", "pass", "wincred", "awssecrets"
	Backend BackendType

	// Pass-specific
	StorePath string // Default: ~/.password-store
	Prefix    string // Default: "dotfiles"

	// Session management
	SessionFile string // Where to cache session token
	SessionTTL  int    // How long to cache in seconds (default: 1800 / 30m)

	// Backend-specific options
	Options map[string]string
}

// BackendFactory creates a backend from configuration.
type BackendFactory func(cfg Config) (Backend, error)

var (
	backendFactories = make(map[BackendType]BackendFactory)
	mu               sync.RWMutex
)

// RegisterBackend registers a backend factory function.
// Backend implementations should call this in their init() function.
func RegisterBackend(backendType BackendType, factory BackendFactory) {
	mu.Lock()
	defer mu.Unlock()
	backendFactories[backendType] = factory
}

// New creates a new vault backend based on configuration.
// The backend package must be imported for the backend to be available.
// Example: import _ "github.com/blackwell-systems/vaultmux/backends/pass"
func New(cfg Config) (Backend, error) {
	// Apply defaults
	if cfg.SessionTTL == 0 {
		cfg.SessionTTL = 1800 // 30 minutes
	}

	mu.RLock()
	factory, ok := backendFactories[cfg.Backend]
	mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown backend: %s (did you import the backend package?)", cfg.Backend)
	}

	return factory(cfg)
}

// MustNew creates a backend or panics. Use in init() only.
func MustNew(cfg Config) Backend {
	b, err := New(cfg)
	if err != nil {
		panic(err)
	}
	return b
}
