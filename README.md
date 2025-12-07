# Vaultmux

> **Unified interface for multiple secret management backends**

[![Blackwell Systems™](https://raw.githubusercontent.com/blackwell-systems/blackwell-docs-theme/main/badge-trademark.svg)](https://github.com/blackwell-systems)
[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://go.dev/)
[![Version](https://img.shields.io/github/v/release/blackwell-systems/vaultmux)](https://github.com/blackwell-systems/vaultmux/releases)
[![CI](https://github.com/blackwell-systems/vaultmux/workflows/CI/badge.svg)](https://github.com/blackwell-systems/vaultmux/actions)
[![codecov](https://codecov.io/gh/blackwell-systems/vaultmux/branch/main/graph/badge.svg)](https://codecov.io/gh/blackwell-systems/vaultmux)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Sponsor](https://img.shields.io/badge/Sponsor-Buy%20Me%20a%20Coffee-yellow?logo=buy-me-a-coffee&logoColor=white)](https://buymeacoffee.com/blackwellsystems)

Vaultmux is a Go library that provides a unified interface for interacting with multiple secret management backends. Write your code once and support Bitwarden, 1Password, and pass (the standard Unix password manager) with the same API.

## Features

- **Unified API** - Single interface works with any backend
- **Zero External Dependencies** - Only Go stdlib; backends use their own CLIs
- **Context Support** - All operations accept `context.Context` for cancellation/timeout
- **Session Caching** - Avoid repeated authentication prompts
- **Type-Safe** - Full static typing with Go interfaces
- **Testable** - Includes mock backend for unit testing (89%+ core coverage)

## Supported Backends

| Backend | CLI Tool | Features |
|---------|----------|----------|
| **Bitwarden** | `bw` | Session tokens, folders, sync |
| **1Password** | `op` | Session tokens, vaults, auto-sync |
| **pass** | `pass` + `gpg` | Git-based, directories, offline |

## Installation

```bash
go get github.com/blackwell-systems/vaultmux
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/blackwell-systems/vaultmux"
)

func main() {
    ctx := context.Background()

    // Create backend (auto-detects ~/.password-store)
    backend, err := vaultmux.New(vaultmux.Config{
        Backend: vaultmux.BackendPass,
        Prefix:  "myapp",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer backend.Close()

    // Initialize (checks CLI availability)
    if err := backend.Init(ctx); err != nil {
        log.Fatal(err)
    }

    // Authenticate (no-op for pass, interactive for Bitwarden/1Password)
    session, err := backend.Authenticate(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // Store a secret
    err = backend.CreateItem(ctx, "API-Key", "sk-secret123", session)
    if err != nil {
        log.Fatal(err)
    }

    // Retrieve it
    secret, err := backend.GetNotes(ctx, "API-Key", session)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Secret:", secret)
}
```

## Configuration

```go
config := vaultmux.Config{
    // Backend type: "bitwarden", "1password", or "pass"
    Backend: vaultmux.BackendPass,

    // Pass-specific
    StorePath: "/custom/path/.password-store", // Default: ~/.password-store
    Prefix:    "myapp",                        // Default: "dotfiles"

    // Session management (for Bitwarden/1Password)
    SessionFile: "/tmp/.vault-session", // Where to cache tokens
    SessionTTL:  1800,                  // Seconds (default: 30 minutes)

    // Backend-specific options
    Options: map[string]string{
        // Backend-specific key-value pairs
    },
}

backend, err := vaultmux.New(config)
```

## Usage Examples

### With Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

notes, err := backend.GetNotes(ctx, "SSH-Config", session)
if err != nil {
    if ctx.Err() == context.DeadlineExceeded {
        log.Println("Operation timed out")
    }
    return err
}
```

### Error Handling

```go
notes, err := backend.GetNotes(ctx, "API-Key", session)
if err != nil {
    // Check for specific errors
    if errors.Is(err, vaultmux.ErrNotFound) {
        // Item doesn't exist - create it
        return backend.CreateItem(ctx, "API-Key", content, session)
    }
    if errors.Is(err, vaultmux.ErrSessionExpired) {
        // Re-authenticate and retry
        session, _ = backend.Authenticate(ctx)
        return backend.GetNotes(ctx, "API-Key", session)
    }
    return err
}
```

### Backend Auto-Detection

```go
func DetectBackend() vaultmux.BackendType {
    if _, err := exec.LookPath("bw"); err == nil {
        return vaultmux.BackendBitwarden
    }
    if _, err := exec.LookPath("op"); err == nil {
        return vaultmux.BackendOnePassword
    }
    if _, err := exec.LookPath("pass"); err == nil {
        return vaultmux.BackendPass
    }
    return "" // No backend available
}
```

### List and Sync

```go
// Pull latest from server (no-op for pass)
if err := backend.Sync(ctx, session); err != nil {
    return err
}

// List all items
items, err := backend.ListItems(ctx, session)
if err != nil {
    return err
}

for _, item := range items {
    fmt.Printf("- %s (type: %s)\n", item.Name, item.Type)
}
```

### Working with Locations (Folders/Vaults)

```go
// List locations (folders in Bitwarden, vaults in 1Password, directories in pass)
locations, err := backend.ListLocations(ctx, session)
if err != nil {
    return err
}

// Create a location
err = backend.CreateLocation(ctx, "work-secrets", session)
if err != nil {
    return err
}

// List items in a specific location
items, err := backend.ListItemsInLocation(ctx, "folder", "work-secrets", session)
```

## Testing

Vaultmux includes a mock backend for unit testing:

```go
import (
    "testing"
    "github.com/blackwell-systems/vaultmux/mock"
)

func TestMyCode(t *testing.T) {
    // Create mock backend
    backend := mock.New()

    // Pre-populate with test data
    backend.SetItem("test-key", "test-value")

    // Test error conditions
    backend.GetError = errors.New("simulated error")

    // Run your tests...
}
```

### Integration Tests

To run integration tests against real backends:

```bash
# Install pass and set up a test store
pass init test@example.com

# Run tests
VAULTMUX_TEST_PASS=1 go test -tags=integration ./...

# Or for other backends
VAULTMUX_TEST_BITWARDEN=1 go test -tags=integration ./...
VAULTMUX_TEST_1PASSWORD=1 go test -tags=integration ./...
```

## Architecture

See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed design documentation.

**Key Design Principles:**

1. **Backend CLIs are the source of truth** - We shell out to `bw`, `op`, `pass` rather than reimplementing protocols
2. **Fail fast, fail clearly** - Explicit errors over silent failures
3. **No global state** - All state lives in Backend/Session structs
4. **Functional options** - Extensible configuration without breaking changes

## API Reference

### Core Interfaces

```go
// Backend represents a secret storage backend
type Backend interface {
    Name() string
    Init(ctx context.Context) error
    Close() error

    IsAuthenticated(ctx context.Context) bool
    Authenticate(ctx context.Context) (Session, error)
    Sync(ctx context.Context, session Session) error

    GetItem(ctx context.Context, name string, session Session) (*Item, error)
    GetNotes(ctx context.Context, name string, session Session) (string, error)
    ItemExists(ctx context.Context, name string, session Session) (bool, error)
    ListItems(ctx context.Context, session Session) ([]*Item, error)

    CreateItem(ctx context.Context, name, content string, session Session) error
    UpdateItem(ctx context.Context, name, content string, session Session) error
    DeleteItem(ctx context.Context, name string, session Session) error

    LocationManager
}

// Session represents an authenticated session
type Session interface {
    Token() string
    IsValid(ctx context.Context) bool
    Refresh(ctx context.Context) error
    ExpiresAt() time.Time
}

// LocationManager handles organizational units
type LocationManager interface {
    ListLocations(ctx context.Context, session Session) ([]string, error)
    LocationExists(ctx context.Context, name string, session Session) (bool, error)
    CreateLocation(ctx context.Context, name string, session Session) error
    ListItemsInLocation(ctx context.Context, locType, locValue string, session Session) ([]*Item, error)
}
```

### Common Errors

```go
var (
    ErrNotFound            = errors.New("item not found")
    ErrAlreadyExists       = errors.New("item already exists")
    ErrNotAuthenticated    = errors.New("not authenticated")
    ErrSessionExpired      = errors.New("session expired")
    ErrBackendNotInstalled = errors.New("backend CLI not installed")
    ErrBackendLocked       = errors.New("vault is locked")
    ErrPermissionDenied    = errors.New("permission denied")
)
```

## Backend Comparison

| Feature | Bitwarden | 1Password | pass |
|---------|-----------|-----------|------|
| **CLI Tool** | `bw` | `op` | `pass` |
| **Auth Method** | Email/password + 2FA | Account + biometrics | GPG key |
| **Session Duration** | Until lock | 30 minutes | GPG agent TTL |
| **Sync** | `bw sync` | Automatic | `pass git pull/push` |
| **Offline Mode** | Yes (cached) | Limited | Yes (local files) |
| **Folders** | Yes (folderId) | Vaults | Directories |
| **Sharing** | Organizations | Vaults | Git repos |
| **Free Tier** | Yes | No | Yes (FOSS) |
| **Self-Host** | Yes (Vaultwarden) | No | Yes (any git host) |

### When to Use Each

**Bitwarden:**
- Team/organization use
- Cross-platform sync needed
- Self-hosting preferred (Vaultwarden)

**1Password:**
- Enterprise environments
- Biometric auth important
- Watchtower/security features needed

**pass:**
- Unix power users
- Git-based workflow
- Minimal dependencies preferred
- Full offline support needed

## Requirements

Each backend requires its CLI tool to be installed:

```bash
# Bitwarden
npm install -g @bitwarden/cli

# 1Password
# See: https://1password.com/downloads/command-line/

# pass
apt-get install pass  # Debian/Ubuntu
brew install pass      # macOS
```

## Security Considerations

1. **Session tokens are sensitive** - Stored with mode 0600, cleared on exit
2. **GPG agent passphrase caching** - Configure appropriate timeout
3. **CLI output may contain secrets** - Don't log full command output
4. **Context cancellation** - Ensure partial operations are safe
5. **Concurrent access** - Session refresh uses mutex protection

## Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass: `go test ./...`
5. Run linter: `golangci-lint run`
6. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) for details

## Credits

Created as part of the [blackwell-systems/dotfiles](https://github.com/blackwell-systems/dotfiles) project.

## Related Projects

- [Bitwarden CLI](https://github.com/bitwarden/clients)
- [1Password CLI](https://developer.1password.com/docs/cli/)
- [pass](https://www.passwordstore.org/)
