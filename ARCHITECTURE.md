# Vaultmux Architecture Document

> **Module:** `github.com/blackwell-systems/vaultmux`
> **Status:** Production
> **Author:** Blackwell Systems
> **Created:** 2025-12-07

---

## Executive Summary

Vaultmux is a Go library that provides a unified interface for interacting with multiple secret management backends. It abstracts away the differences between Bitwarden, 1Password, pass (Unix password manager), Windows Credential Manager, and AWS Secrets Manager, allowing applications to work with any supported backend through a single API.

**Key Features:**
- Unified `Backend` interface for all secret managers
- Session management with caching and refresh
- Context-aware operations with timeout support
- Location/folder management across backends
- Multiple integration patterns: CLI wrappers, native SDKs, OS APIs
- Minimal dependencies (stdlib + backend-specific SDKs only)

---

## Table of Contents

1. [Design Goals](#1-design-goals)
2. [Architecture Overview](#2-architecture-overview)
3. [Core Interfaces](#3-core-interfaces)
4. [Authentication Flow](#4-authentication-flow)
5. [Session Management](#5-session-management)
6. [CRUD Operations](#6-crud-operations)
7. [Backend Registration](#7-backend-registration)
8. [Error Handling](#8-error-handling)
9. [Testing Strategy](#9-testing-strategy)
10. [Backend Comparison](#10-backend-comparison)

---

## 1. Design Goals

### 1.1 Primary Goals

| Goal | Description |
|------|-------------|
| **Unified API** | Single interface works with any backend |
| **Minimal Dependencies** | Core has zero dependencies; backends add only what they need (CLIs or SDKs) |
| **Context Support** | All operations accept `context.Context` for cancellation/timeout |
| **Session Caching** | Avoid repeated authentication prompts |
| **Testability** | Mock backend for unit testing |

### 1.2 Non-Goals

| Non-Goal | Rationale |
|----------|-----------|
| GUI/TUI | Library only; consumers build their own UI |
| Backend installation | Users install `bw`, `op`, `pass` themselves |
| Encryption | Delegated to backend implementations |
| Key generation | Out of scope; use backend tools |

### 1.3 Design Principles

1. **Backends integrate natively** - CLI backends shell out to `bw`, `op`, `pass`; SDK backends use native clients (AWS SDK, Azure SDK)
2. **Fail fast, fail clearly** - Explicit errors over silent failures
3. **No global state** - All state lives in Backend/Session structs
4. **Functional options** - Extensible configuration without breaking changes
5. **Interface universality** - Same `Backend` interface works for CLI wrappers, OS APIs, and SDK clients

---

## 2. Architecture Overview

### 2.1 Package Structure

```
github.com/blackwell-systems/vaultmux/
├── vaultmux.go           # Core types: Backend, Session, Item, errors
├── factory.go            # New() factory, backend registration
├── session.go            # Session interface, caching logic
├── errors.go             # Typed errors: ErrNotFound, ErrAuth, etc.
│
├── backends/
│   ├── bitwarden/
│   │   ├── bitwarden.go  # Bitwarden CLI backend
│   │   └── session.go    # BW session management
│   │
│   ├── onepassword/
│   │   ├── onepassword.go # 1Password CLI backend
│   │   └── session.go     # OP session management
│   │
│   └── pass/
│       └── pass.go        # pass (GPG-based) backend
│
├── mock/
│   └── mock.go            # In-memory mock for testing
│
└── internal/
    └── exec/
        └── exec.go        # Command execution helpers
```

### 2.2 Component Architecture

```mermaid
graph TB
    subgraph Consumer["Consumer Application"]
        App[Application Code]
    end

    subgraph VaultmuxCore["Vaultmux Core"]
        Factory[Factory<br/>New Config]
        Backend[Backend Interface]
        Session[Session Interface]
        Cache[SessionCache]
    end

    subgraph Backends["Backend Implementations"]
        BW[Bitwarden<br/>Backend]
        OP[1Password<br/>Backend]
        Pass[pass<br/>Backend]
        Mock[Mock<br/>Backend]
    end

    subgraph CLI["External CLI Tools"]
        BWCLI[bw<br/>Bitwarden CLI]
        OPCLI[op<br/>1Password CLI]
        PassCLI[pass + GPG]
    end

    App -->|New Config| Factory
    Factory -->|Create| Backend
    Backend -->|Authenticate| Session
    Session -.->|Cache| Cache

    Backend -.->|implements| BW
    Backend -.->|implements| OP
    Backend -.->|implements| Pass
    Backend -.->|implements| Mock

    BW -->|exec| BWCLI
    OP -->|exec| OPCLI
    Pass -->|exec| PassCLI

    classDef coreStyle fill:#2d7dd2,stroke:#4a9eff,stroke-width:2px,color:#e0e0e0
    classDef backendStyle fill:#3a3a3a,stroke:#6eb5ff,stroke-width:1px,color:#e0e0e0
    classDef cliStyle fill:#1a1a1a,stroke:#4a9eff,stroke-width:1px,color:#e0e0e0

    class Factory,Backend,Session,Cache coreStyle
    class BW,OP,Pass,Mock backendStyle
    class BWCLI,OPCLI,PassCLI cliStyle
```

### 2.3 Data Flow

```mermaid
sequenceDiagram
    participant Consumer
    participant Factory
    participant Backend
    participant Session
    participant CLI as Backend CLI

    Consumer->>Factory: New(Config)
    Factory->>Factory: Apply Defaults
    Factory->>Factory: Lookup Backend
    Factory->>Backend: factory(cfg)
    Backend-->>Factory: backend instance
    Factory-->>Consumer: backend

    Consumer->>Backend: Init(ctx)
    Backend->>CLI: Check CLI installed
    CLI-->>Backend: exists
    Backend-->>Consumer: nil (success)

    Consumer->>Backend: Authenticate(ctx)
    Backend->>Session: Check cached
    alt Cache Valid
        Session-->>Backend: cached token
    else Cache Invalid/Missing
        Backend->>CLI: bw unlock / op signin
        CLI-->>Backend: session token
        Backend->>Session: Cache token
    end
    Backend-->>Consumer: Session

    Consumer->>Backend: GetNotes(ctx, name, session)
    Backend->>CLI: bw get item / op item get
    CLI-->>Backend: item JSON
    Backend-->>Consumer: notes string
```

---

## 3. Core Interfaces

### 3.1 Backend Interface

```go
// Backend represents a secret storage backend.
type Backend interface {
    // Metadata
    Name() string

    // Lifecycle
    Init(ctx context.Context) error
    Close() error

    // Authentication
    IsAuthenticated(ctx context.Context) bool
    Authenticate(ctx context.Context) (Session, error)

    // Sync (pull latest from server)
    Sync(ctx context.Context, session Session) error

    // Item Operations (CRUD)
    GetItem(ctx context.Context, name string, session Session) (*Item, error)
    GetNotes(ctx context.Context, name string, session Session) (string, error)
    ItemExists(ctx context.Context, name string, session Session) (bool, error)
    ListItems(ctx context.Context, session Session) ([]*Item, error)

    // Mutations
    CreateItem(ctx context.Context, name, content string, session Session) error
    UpdateItem(ctx context.Context, name, content string, session Session) error
    DeleteItem(ctx context.Context, name string, session Session) error

    // Location Management (folders/vaults)
    LocationManager
}
```

### 3.2 Session Interface

```go
// Session represents an authenticated session.
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
```

### 3.3 Item Structure

```go
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
    ItemTypeSecureNote ItemType = iota
    ItemTypeLogin
    ItemTypeSSHKey
    ItemTypeIdentity
    ItemTypeCard
)
```

---

## 4. Authentication Flow

```mermaid
flowchart TD
    Start([Authenticate ctx]) --> CheckCache{Cached<br/>Session<br/>Exists?}

    CheckCache -->|Yes| ValidateCache{Session<br/>Valid?}
    CheckCache -->|No| CheckStatus

    ValidateCache -->|Yes| ReturnCached[Return Cached Session]
    ValidateCache -->|No| CheckStatus

    CheckStatus[Check CLI Status] --> IsLoggedIn{Logged<br/>In?}

    IsLoggedIn -->|No| ErrorNotLoggedIn[Error: Not Logged In<br/>Hint: bw login]
    IsLoggedIn -->|Yes| IsLocked{Vault<br/>Locked?}

    IsLocked -->|Yes| PromptPassword[Prompt for Password]
    IsLocked -->|No| GetToken

    PromptPassword --> ExecuteUnlock[Execute unlock CLI]
    ExecuteUnlock --> GetToken[Get Session Token]

    GetToken --> CacheToken[Cache Token<br/>TTL: 30 min]
    CacheToken --> CreateSession[Create Session Object]
    CreateSession --> ReturnSession[Return Session]

    ReturnCached --> End([Session])
    ErrorNotLoggedIn --> End
    ReturnSession --> End

    style Start fill:#2d7dd2,stroke:#4a9eff,color:#e0e0e0
    style End fill:#2d7dd2,stroke:#4a9eff,color:#e0e0e0
    style CheckCache fill:#3a3a3a,stroke:#6eb5ff,color:#e0e0e0
    style ValidateCache fill:#3a3a3a,stroke:#6eb5ff,color:#e0e0e0
    style IsLoggedIn fill:#3a3a3a,stroke:#6eb5ff,color:#e0e0e0
    style IsLocked fill:#3a3a3a,stroke:#6eb5ff,color:#e0e0e0
```

### 4.1 Authentication States

| State | Description | CLI Check | Action |
|-------|-------------|-----------|--------|
| **Unauthenticated** | Never logged in | `bw status` → unauthenticated | Prompt: `bw login` |
| **Locked** | Logged in but locked | `bw status` → locked | Execute: `bw unlock` |
| **Unlocked** | Active session | `bw unlock --check` → success | Use cached token |
| **Expired** | Session expired | Token fails validation | Re-unlock or refresh |

---

## 5. Session Management

### 5.1 Session Caching Strategy

```mermaid
graph LR
    subgraph Memory["In-Memory"]
        Sess[Session Object]
    end

    subgraph Disk["~/.config/vaultmux/"]
        Cache[.bw-session<br/>JSON File]
    end

    subgraph CacheData["Cache Structure"]
        Token[Token: string]
        Created[Created: time]
        Expires[Expires: time]
        Backend[Backend: string]
    end

    Sess -->|Save| Cache
    Cache -->|Load| Sess
    Cache -.->|Contains| CacheData

    style Sess fill:#2d7dd2,stroke:#4a9eff,color:#e0e0e0
    style Cache fill:#3a3a3a,stroke:#6eb5ff,color:#e0e0e0
    style Token,Created,Expires,Backend fill:#1a1a1a,stroke:#4a9eff,color:#e0e0e0
```

### 5.2 Cache Operations

```go
// SessionCache handles session persistence to disk.
type SessionCache struct {
    path string
    ttl  time.Duration
}

// Load reads a cached session from disk.
// Returns nil if cache doesn't exist, is invalid, or expired.
func (c *SessionCache) Load() (*CachedSession, error)

// Save writes a session to disk with restricted permissions (0600).
func (c *SessionCache) Save(token, backend string) error

// Clear removes the cached session.
func (c *SessionCache) Clear() error
```

### 5.3 Auto-Refresh Pattern

```mermaid
sequenceDiagram
    participant App
    participant AutoRefresh as AutoRefreshSession
    participant Inner as Inner Session
    participant Backend

    App->>AutoRefresh: Token()
    AutoRefresh->>Inner: IsValid(ctx)?

    alt Session Valid
        Inner-->>AutoRefresh: true
        AutoRefresh->>Inner: Token()
        Inner-->>AutoRefresh: token
        AutoRefresh-->>App: token
    else Session Expired
        Inner-->>AutoRefresh: false
        AutoRefresh->>Inner: Refresh(ctx)
        Inner->>Backend: Re-authenticate
        Backend-->>Inner: new token
        Inner-->>AutoRefresh: success
        AutoRefresh->>Inner: Token()
        Inner-->>AutoRefresh: new token
        AutoRefresh-->>App: new token
    end
```

---

## 6. CRUD Operations

### 6.1 GetNotes Operation Flow

```mermaid
flowchart TD
    Start([GetNotes ctx, name, session]) --> ValidateSession{Session<br/>Valid?}

    ValidateSession -->|No| ReturnAuthError[Return: ErrNotAuthenticated]
    ValidateSession -->|Yes| ExecuteCLI[Execute CLI Command<br/>bw get item name]

    ExecuteCLI --> CheckExit{Exit<br/>Code}

    CheckExit -->|0| ParseJSON[Parse JSON Response]
    CheckExit -->|Non-zero| CheckError{Error<br/>Type?}

    CheckError -->|Not Found| ReturnNotFound[Return: ErrNotFound]
    CheckError -->|Auth Failed| ReturnExpired[Return: ErrSessionExpired]
    CheckError -->|Other| ReturnError[Return: Wrapped Error]

    ParseJSON --> ExtractNotes[Extract notes Field]
    ExtractNotes --> ReturnNotes[Return: notes, nil]

    ReturnAuthError --> End([notes, error])
    ReturnNotFound --> End
    ReturnExpired --> End
    ReturnError --> End
    ReturnNotes --> End

    style Start fill:#2d7dd2,stroke:#4a9eff,color:#e0e0e0
    style End fill:#2d7dd2,stroke:#4a9eff,color:#e0e0e0
    style ValidateSession fill:#3a3a3a,stroke:#6eb5ff,color:#e0e0e0
    style CheckExit fill:#3a3a3a,stroke:#6eb5ff,color:#e0e0e0
    style CheckError fill:#3a3a3a,stroke:#6eb5ff,color:#e0e0e0
```

### 6.2 CreateItem Operation

```mermaid
sequenceDiagram
    participant App
    participant Backend
    participant CLI as Backend CLI
    participant Server as Vault Server

    App->>Backend: CreateItem(ctx, name, content, session)
    Backend->>Backend: Validate session
    Backend->>Backend: Check item exists

    alt Item Already Exists
        Backend-->>App: ErrAlreadyExists
    else Item Doesn't Exist
        Backend->>Backend: Build item JSON
        Backend->>CLI: create item --template json
        CLI->>Server: Upload encrypted item
        Server-->>CLI: Success + item ID
        CLI-->>Backend: Exit code 0
        Backend-->>App: nil (success)
    end
```

---

## 7. Backend Registration

### 7.1 Registration Pattern

```mermaid
graph TD
    subgraph Import["Backend Package Import"]
        InitFunc[init Function]
    end

    subgraph Factory["Factory Registry"]
        Registry[map<br/>BackendType → Factory]
        Register[RegisterBackend]
    end

    subgraph Creation["Backend Creation"]
        NewCall[New Config]
        Lookup[Lookup Factory]
        Create[Call Factory Function]
    end

    InitFunc -->|Calls| Register
    Register -->|Stores in| Registry
    NewCall -->|Queries| Registry
    Registry -->|Returns| Lookup
    Lookup -->|Invokes| Create

    style InitFunc fill:#2d7dd2,stroke:#4a9eff,color:#e0e0e0
    style Registry fill:#3a3a3a,stroke:#6eb5ff,color:#e0e0e0
    style Create fill:#2d7dd2,stroke:#4a9eff,color:#e0e0e0
```

### 7.2 Example Backend Registration

```go
// In backends/bitwarden/bitwarden.go
package bitwarden

import "github.com/blackwell-systems/vaultmux"

func init() {
    vaultmux.RegisterBackend(vaultmux.BackendBitwarden, func(cfg vaultmux.Config) (vaultmux.Backend, error) {
        return New(cfg.Options, cfg.SessionFile)
    })
}
```

### 7.3 Consumer Usage

```go
// Consumer application
import (
    "github.com/blackwell-systems/vaultmux"
    _ "github.com/blackwell-systems/vaultmux/backends/bitwarden"  // Register via init()
)

func main() {
    backend, err := vaultmux.New(vaultmux.Config{
        Backend: vaultmux.BackendBitwarden,
    })
    // backend.init() was called during import
}
```

---

## 8. Error Handling

### 8.1 Error Types

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

### 8.2 Error Wrapping

```mermaid
graph LR
    CLIError[CLI Error<br/>exit code 1] --> WrapError[WrapError<br/>backend, operation, item]
    WrapError --> VaultmuxError[Vaultmux Error<br/>with context]
    VaultmuxError --> Consumer[Consumer<br/>errors.Is check]

    style CLIError fill:#1a1a1a,stroke:#4a9eff,color:#e0e0e0
    style WrapError fill:#3a3a3a,stroke:#6eb5ff,color:#e0e0e0
    style VaultmuxError fill:#2d7dd2,stroke:#4a9eff,color:#e0e0e0
    style Consumer fill:#2d7dd2,stroke:#4a9eff,color:#e0e0e0
```

```go
// WrapError wraps CLI errors with context.
func WrapError(backend, operation, item string, err error) error {
    return fmt.Errorf("%s %s %q: %w", backend, operation, item, err)
}

// Consumer code
notes, err := backend.GetNotes(ctx, "SSH-Config", session)
if err != nil {
    if errors.Is(err, vaultmux.ErrNotFound) {
        // Item doesn't exist - create it
    }
    if errors.Is(err, vaultmux.ErrSessionExpired) {
        // Re-authenticate
    }
    return err
}
```

---

## 9. Testing Strategy

### 9.1 Test Pyramid

```mermaid
graph TD
    subgraph Integration["Integration Tests<br/>Real CLI"]
        IntTests[Test with pass<br/>in test environment]
    end

    subgraph Unit["Unit Tests<br/>Mock Backend"]
        UnitTests[89%+ Coverage<br/>No external deps]
    end

    subgraph Examples["Examples"]
        ExampleTests[Runnable Examples<br/>Documentation]
    end

    Integration -.->|Few| Top[Tests]
    Unit -.->|Most| Top
    Examples -.->|Some| Top

    style Integration fill:#3a3a3a,stroke:#6eb5ff,color:#e0e0e0
    style Unit fill:#2d7dd2,stroke:#4a9eff,color:#e0e0e0
    style Examples fill:#3a3a3a,stroke:#6eb5ff,color:#e0e0e0
```

### 9.2 Mock Backend

```go
// Mock backend for unit testing
import "github.com/blackwell-systems/vaultmux/mock"

func TestMyCode(t *testing.T) {
    backend := mock.New()

    // Pre-populate with test data
    backend.SetItem("test-key", "test-value")

    // Test error conditions
    backend.GetError = errors.New("simulated error")

    // Your tests here...
}
```

---

## 10. Backend Comparison

### 10.1 Feature Matrix

| Feature | Bitwarden | 1Password | pass | Windows Cred Mgr | AWS Secrets Manager |
|---------|-----------|-----------|------|------------------|---------------------|
| **Integration** | CLI (`bw`) | CLI (`op`) | CLI (`pass`) | PowerShell | SDK (aws-sdk-go-v2) |
| **Auth Method** | Email/password + 2FA | Account + biometrics | GPG key | OS-level / Windows Hello | IAM credentials |
| **Session Duration** | Until lock | 30 minutes | GPG agent TTL | OS-managed | Long-lived (IAM) |
| **Sync** | `bw sync` | Automatic | `pass git pull/push` | None (local only) | Always synchronized |
| **Offline Mode** | Yes (cached) | Limited | Yes (local files) | Yes (always local) | No (requires AWS API) |
| **Folders** | Yes (folderId) | Vaults | Directories | No (flat namespace) | Prefix + tags |
| **Sharing** | Organizations | Vaults | Git repos | Windows user account | IAM policies |
| **Free Tier** | Yes | No | Yes (FOSS) | Yes (built-in) | No (~$0.40/secret/month) |
| **Self-Host** | Yes (Vaultwarden) | No | Yes (any git host) | N/A (local OS) | No (AWS only) |
| **Platform** | All | All | Unix | Windows only | All |
| **Versioning** | No | No | Via git | No | Automatic (built-in) |
| **Rotation** | Manual | Manual | Manual | Manual | Automatic (configurable) |
| **Audit Logging** | Self-hosted only | Enterprise only | Via git log | No | Built-in (CloudTrail) |

### 10.2 Implementation Differences

```mermaid
graph TB
    subgraph BW["Bitwarden"]
        BWSess[Session Token]
        BWSync[bw sync]
        BWFolder[Folders by ID]
    end

    subgraph OP["1Password"]
        OPSess[Biometric + Session]
        OPSync[Auto-sync]
        OPVault[Vaults]
    end

    subgraph P["pass"]
        PSess[No Session<br/>GPG agent]
        PSync[git pull/push]
        PDir[Directories]
    end

    subgraph WC["Windows Cred Mgr"]
        WCSess[No Session<br/>OS Auth]
        WCSync[None - local only]
        WCFolder[No folders]
    end

    subgraph AWS["AWS Secrets Manager"]
        AWSSess[IAM Credentials<br/>SDK-managed]
        AWSSync[Always synchronized]
        AWSFolder[Prefix + Tags]
        AWSVer[Automatic versioning]
    end

    Backend[Backend Interface] -.->|implements| BW
    Backend -.->|implements| OP
    Backend -.->|implements| P
    Backend -.->|implements| WC
    Backend -.->|implements| AWS

    style Backend fill:#2d7dd2,stroke:#4a9eff,color:#e0e0e0
    style BW,OP,P,WC,AWS fill:#3a3a3a,stroke:#6eb5ff,color:#e0e0e0
```

### 10.3 When to Use Each Backend

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
- Windows development environments
- No external tools or dependencies
- Windows Hello / biometric auth
- Single-machine secrets (no sync needed)
- Quick setup for Windows-only projects

**AWS Secrets Manager:**
- Applications running on AWS (EC2, ECS, Lambda, EKS)
- Automatic secret rotation needed (databases, API keys)
- IAM-based access control and fine-grained permissions
- Audit logging requirements (CloudTrail integration)
- Multi-region replication and disaster recovery
- Versioning and rollback capabilities
- Integration with AWS services (RDS, Redshift, DocumentDB)
- Teams already invested in AWS ecosystem

---

## Conclusion

Vaultmux provides a clean abstraction over multiple secret management backends, allowing applications to switch backends with minimal code changes. The architecture prioritizes simplicity, testability, and reliability by delegating backend-specific complexity to their respective CLIs rather than reimplementing protocols.

**Key Architectural Decisions:**
1. **Interface-first design** - Clear contracts between layers
2. **CLI delegation** - Leverage battle-tested implementations
3. **Session caching** - Balance security and UX
4. **Context propagation** - Proper cancellation and timeout support
5. **Type-safe errors** - Clear error handling patterns

**Extensibility:**
New backends can be added by implementing the `Backend` interface and registering via `RegisterBackend()` in the package's `init()` function. See [EXTENDING.md](EXTENDING.md) for details.

---

**Related Documentation:**
- [README.md](README.md) - Quick start and usage
- [EXTENDING.md](EXTENDING.md) - Adding new backends
- [Go Package Documentation](https://pkg.go.dev/github.com/blackwell-systems/vaultmux)
