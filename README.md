# Vaultmux

> **The definitive Go library for multi-vault secret management**

[![Blackwell Systems™](https://raw.githubusercontent.com/blackwell-systems/blackwell-docs-theme/main/badge-trademark.svg)](https://github.com/blackwell-systems) 
[![Go Reference](https://pkg.go.dev/badge/github.com/blackwell-systems/vaultmux.svg)](https://pkg.go.dev/github.com/blackwell-systems/vaultmux) 
[![Go Version](https://img.shields.io/badge/go-1.23+-blue.svg)](https://go.dev/) 
[![Version](https://img.shields.io/github/v/release/blackwell-systems/vaultmux)](https://github.com/blackwell-systems/vaultmux/releases) 

[![CI](https://github.com/blackwell-systems/vaultmux/workflows/CI/badge.svg)](https://github.com/blackwell-systems/vaultmux/actions) [![codecov](https://codecov.io/gh/blackwell-systems/vaultmux/branch/main/graph/badge.svg)](https://codecov.io/gh/blackwell-systems/vaultmux)
[![Sponsor](https://img.shields.io/badge/Sponsor-Buy%20Me%20a%20Coffee-yellow?logo=buy-me-a-coffee&logoColor=white)](https://buymeacoffee.com/blackwellsystems)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE) 

Vaultmux provides a unified interface for interacting with multiple secret management systems. Write your code once and support Bitwarden, 1Password, pass, Windows Credential Manager, AWS Secrets Manager, Google Cloud Secret Manager, and Azure Key Vault with the same API. Switch backends with zero code changes—perfect for multi-cloud deployments, cross-platform applications, and teams migrating between secret management solutions.

## Features

- **Unified API** - Single interface works with any backend
- **Minimal Dependencies** - Core has zero dependencies; AWS/GCP backends use official SDKs
- **Context Support** - All operations accept `context.Context` for cancellation/timeout
- **Session Caching** - Avoid repeated authentication prompts
- **Type-Safe** - Full static typing with Go interfaces
- **Testable** - Includes mock backend for unit testing (98%+ core coverage)

## Supported Backends

| Backend | Integration | Features | Platform |
|---------|-------------|----------|----------|
| **Bitwarden** | CLI (`bw`) | Session tokens, folders, sync | All |
| **1Password** | CLI (`op`) | Session tokens, vaults, auto-sync | All |
| **pass** | CLI (`pass` + `gpg`) | Git-based, directories, offline | Unix |
| **Windows Credential Manager** | PowerShell | OS-level auth, Windows Hello | Windows |
| **AWS Secrets Manager** | SDK (aws-sdk-go-v2) | IAM auth, versioning, rotation | All |
| **Google Cloud Secret Manager** | SDK (cloud.google.com/go) | ADC auth, auto-versioning, labels | All |
| **Azure Key Vault** | SDK (azure-sdk-for-go) | Azure AD auth, HSM-backed, RBAC | All |

## Why Vaultmux?

**The Problem**: Applications that hardcode a single secret management backend lock users into one workflow. Supporting multiple backends with different APIs creates maintenance nightmares—you end up writing the same logic multiple times with different method signatures, error handling, and authentication patterns.

**The Solution**: Vaultmux provides a unified API that eliminates repetition. Write your secret management code once, and it works with any backend. Users choose their preferred system without you writing backend-specific code.

### Common Use Cases

**Multi-Cloud Deployments**
- Deploy to AWS, GCP, and Azure without rewriting secret access logic
- Switch between AWS Secrets Manager, Google Cloud Secret Manager, and Azure Key Vault with a config change

**Cross-Platform Applications**
- Support Windows Credential Manager on Windows, pass on Linux, and 1Password on macOS
- One codebase works everywhere

**Team Flexibility**
- Open source projects where contributors use different secret managers
- Enterprise environments with mixed tooling (some teams on 1Password, others on HashiCorp Vault)
- Migration scenarios (moving from Bitwarden to AWS Secrets Manager)

**Development Workflows**
- Use `pass` locally for development, AWS Secrets Manager in production
- Test with mock backend, deploy with real cloud provider
- No code changes required—just configuration

```go
// Same code works with pass, Bitwarden, 1Password, AWS, GCP, Azure, etc.
backend, _ := vaultmux.New(config)
secret, _ := backend.GetNotes(ctx, "api-key", session)
```

This flexibility means your dotfiles framework (or any secret-dependent application) can support whatever workflows users throw at you—from privacy-focused GPG enthusiasts to enterprise teams using Azure Key Vault—all through one convenient interface.

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
    // Backend type: "bitwarden", "1password", "pass", "wincred", "awssecrets", or "gcpsecrets"
    Backend: vaultmux.BackendPass,

    // Pass-specific
    StorePath: "/custom/path/.password-store", // Default: ~/.password-store
    Prefix:    "myapp",                        // Default: "dotfiles"

    // Session management (for Bitwarden/1Password)
    SessionFile: "/tmp/.vault-session", // Where to cache tokens
    SessionTTL:  1800,                  // Seconds (default: 30 minutes)

    // Backend-specific options
    Options: map[string]string{
        // AWS Secrets Manager:
        "region":   "us-west-2",              // AWS region
        "prefix":   "myapp/",                 // Secret name prefix
        "endpoint": "http://localhost:4566", // LocalStack endpoint (for testing)

        // Google Cloud Secret Manager:
        "project_id": "my-gcp-project",      // GCP project ID (required)
        "prefix":     "myapp-",              // Secret name prefix
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

See [ARCHITECTURE.md](docs/ARCHITECTURE.md) for detailed design documentation.

**Key Design Principles:**

1. **Backends integrate natively** - CLI backends shell out to `bw`, `op`, `pass`; SDK backends use native clients (AWS SDK v2)
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

| Feature | Bitwarden | 1Password | pass | Windows Cred Mgr | AWS Secrets Manager |
|---------|-----------|-----------|------|------------------|---------------------|
| **Integration** | CLI (`bw`) | CLI (`op`) | CLI (`pass`) | PowerShell | SDK (aws-sdk-go-v2) |
| **Auth Method** | Email/password + 2FA | Account + biometrics | GPG key | Windows Hello/PIN | IAM credentials |
| **Session Duration** | Until lock | 30 minutes | GPG agent TTL | OS-managed | Long-lived (IAM keys) |
| **Sync** | `bw sync` | Automatic | `pass git pull/push` | N/A | Always synchronized |
| **Offline Mode** | Yes (cached) | Limited | Yes (local files) | Yes (local only) | No (requires AWS API) |
| **Folders** | Yes (folderId) | Vaults | Directories | Prefix-based | Prefix-based + tags |
| **Sharing** | Organizations | Vaults | Git repos | Machine-local | IAM policies |
| **Free Tier** | Yes | No | Yes (FOSS) | Yes (built-in) | No (~$0.40/secret/month) |
| **Self-Host** | Yes (Vaultwarden) | No | Yes (any git host) | N/A (OS feature) | No (AWS only) |
| **Platform** | All | All | Unix | Windows | All |

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

**Windows Credential Manager:**
- Windows-only deployments
- No external CLI dependencies
- OS-level authentication integration
- Simple credential storage

**AWS Secrets Manager:**
- Applications running on AWS (EC2, ECS, Lambda)
- Automatic secret rotation needed
- IAM-based access control
- Audit logging requirements
- Multi-region redundancy

## Requirements

Each backend has different requirements:

```bash
# Bitwarden
npm install -g @bitwarden/cli

# 1Password
# See: https://1password.com/downloads/command-line/

# pass
apt-get install pass  # Debian/Ubuntu
brew install pass      # macOS

# Windows Credential Manager
# Built into Windows - no installation required!
# Uses PowerShell for credential access

# AWS Secrets Manager
# Requires AWS SDK v2 (automatically installed via go get)
# AWS credentials configured via:
#   - Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)
#   - Shared credentials file (~/.aws/credentials)
#   - IAM instance role (for EC2/ECS)
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

**Versions >= v0.3.2**: Apache License 2.0
**Versions <= v0.3.0**: MIT License

See [LICENSE](LICENSE) for full Apache 2.0 text.

## Brand

The **code** in this repository is licensed under the Apache License 2.0. The **Blackwell Systems™** name and logo are protected trademarks. See [BRAND.md](BRAND.md) for usage guidelines.

## Credits

Created as part of the [blackwell-systems/dotfiles](https://github.com/blackwell-systems/dotfiles) project.

## Related Projects

### Supported Backends
- [Bitwarden CLI](https://github.com/bitwarden/clients) - Open source password manager
- [1Password CLI](https://developer.1password.com/docs/cli/) - Enterprise password management
- [pass](https://www.passwordstore.org/) - Unix password store (GPG-based)
- [Windows Credential Manager](https://docs.microsoft.com/en-us/windows/win32/secauthn/credential-manager) - Native Windows credential storage
- [AWS Secrets Manager](https://aws.amazon.com/secrets-manager/) - Cloud-native secret management

### Similar Projects
- [HashiCorp Vault](https://www.vaultproject.io/) - Enterprise secret management platform
- [Azure Key Vault](https://azure.microsoft.com/en-us/services/key-vault/) - Azure secret management
- [GCP Secret Manager](https://cloud.google.com/secret-manager) - Google Cloud secret management
- [AWS SDK for Go](https://github.com/aws/aws-sdk-go-v2) - Official AWS SDK (used by vaultmux)
