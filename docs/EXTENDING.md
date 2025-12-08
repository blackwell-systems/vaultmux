# Extending Vaultmux: Adding New Backends

This guide walks you through implementing a new vault backend for vaultmux. The library is designed to be extensible, allowing you to add support for any secret management system.

## Table of Contents

1. [Understanding the Architecture](#understanding-the-architecture)
2. [Backend Interface Requirements](#backend-interface-requirements)
3. [Session Management](#session-management)
4. [Implementation Steps](#implementation-steps)
5. [Testing Your Backend](#testing-your-backend)
6. [Registration and Usage](#registration-and-usage)
7. [Best Practices](#best-practices)
8. [Complete Example](#complete-example)

---

## Understanding the Architecture

Vaultmux uses a plugin architecture where backends implement a common interface. This allows applications to switch between vault providers without changing their code.

### Core Components

```
vaultmux/
├── vaultmux.go          # Core interfaces (Backend, Session, Item)
├── factory.go           # Backend registration system
├── session.go           # Session caching utilities
├── errors.go            # Standard error types
└── backends/
    ├── bitwarden/       # Reference: CLI wrapper, token-based, remote sync
    ├── onepassword/     # Reference: CLI wrapper, biometric auth, auto-sync
    ├── pass/            # Reference: CLI wrapper, local GPG-based
    ├── wincred/         # Reference: OS API, platform-specific (Windows only)
    ├── awssecrets/      # Reference: SDK-based, IAM auth, cloud-native
    ├── gcpsecrets/      # Reference: SDK-based, ADC auth, cloud-native
    ├── azurekeyvault/   # Reference: SDK-based, Azure AD auth, HSM-backed
    └── yourbackend/     # Your new backend here
```

### Design Principles

1. **Interface-based**: All backends implement the `Backend` interface
2. **Registration pattern**: Backends self-register using `init()` functions
3. **Context-aware**: All operations accept `context.Context` for cancellation
4. **Error wrapping**: Use `vaultmux.WrapError()` for consistent error messages
5. **Session management**: Backends can leverage built-in session caching

---

## Backend Interface Requirements

Your backend must implement the complete `Backend` interface defined in `vaultmux.go`:

```go
type Backend interface {
    // Metadata
    Name() string

    // Lifecycle
    Init(ctx context.Context) error
    Close() error

    // Authentication
    IsAuthenticated(ctx context.Context) bool
    Authenticate(ctx context.Context) (Session, error)

    // Sync
    Sync(ctx context.Context, session Session) error

    // Item operations
    GetItem(ctx context.Context, name string, session Session) (*Item, error)
    GetNotes(ctx context.Context, name string, session Session) (string, error)
    ItemExists(ctx context.Context, name string, session Session) (bool, error)
    ListItems(ctx context.Context, session Session) ([]*Item, error)

    // Mutations
    CreateItem(ctx context.Context, name, content string, session Session) error
    UpdateItem(ctx context.Context, name, content string, session Session) error
    DeleteItem(ctx context.Context, name string, session Session) error

    // Optional: Location management (folders, vaults, collections)
    LocationManager
}
```

### Method Responsibilities

#### `Name() string`
Returns a unique identifier for your backend (e.g., "bitwarden", "pass", "hashicorp").

#### `Init(ctx context.Context) error`
Performs initialization checks:
- Verify CLI tool is installed
- Check configuration is valid
- Validate credentials file exists
- Return error if prerequisites are missing

#### `Close() error`
Cleanup operations when the backend is no longer needed:
- Close network connections
- Clear sensitive memory
- Release file handles

#### `IsAuthenticated(ctx context.Context) bool`
Check if the user is currently authenticated without prompting.

#### `Authenticate(ctx context.Context) (Session, error)`
Perform authentication and return a session:
- Check cached session first (if applicable)
- Prompt for credentials if needed
- Return a `Session` implementation
- Return error if authentication fails

#### `Sync(ctx context.Context, session Session) error`
Synchronize with remote vault (if supported):
- Pull latest items from server
- Return `nil` if backend doesn't support sync
- Return error if sync fails

#### Item Operations

**GetItem**: Retrieve complete item with metadata
**GetNotes**: Get only the notes/content field (convenience method)
**ItemExists**: Check if item exists without retrieving it
**ListItems**: Return all items (may be filtered by prefix/folder)

#### Mutation Operations

**CreateItem**: Create new item, error if already exists
**UpdateItem**: Update existing item, error if not found
**DeleteItem**: Delete item, error if not found

#### Location Management (Optional)

Implement the `LocationManager` interface if your backend supports organizational units (folders, vaults, collections):

```go
type LocationManager interface {
    ListLocations(ctx context.Context, session Session) ([]string, error)
    LocationExists(ctx context.Context, name string, session Session) (bool, error)
    CreateLocation(ctx context.Context, name string, session Session) error
    ListItemsInLocation(ctx context.Context, locType, locValue string, session Session) ([]*Item, error)
}
```

---

## Session Management

### The Session Interface

Your backend must provide a `Session` implementation:

```go
type Session interface {
    Token() string
    IsValid(ctx context.Context) bool
    Refresh(ctx context.Context) error
    ExpiresAt() time.Time
}
```

### Session Patterns

#### Pattern 1: Token-Based (Bitwarden, 1Password)

Use for backends with explicit session tokens:

```go
type myBackendSession struct {
    token   string
    expires time.Time
    backend *Backend
}

func (s *myBackendSession) Token() string {
    return s.token
}

func (s *myBackendSession) IsValid(ctx context.Context) bool {
    if time.Now().After(s.expires) {
        return false
    }
    // Optionally: verify with backend
    return true
}

func (s *myBackendSession) Refresh(ctx context.Context) error {
    newSession, err := s.backend.Authenticate(ctx)
    if err != nil {
        return err
    }
    s.token = newSession.Token()
    s.expires = newSession.ExpiresAt()
    return nil
}

func (s *myBackendSession) ExpiresAt() time.Time {
    return s.expires
}
```

#### Pattern 2: No-Session (pass, gpg-agent)

Use for backends where authentication is handled by system services:

```go
type passSession struct{}

func (s *passSession) Token() string                     { return "" }
func (s *passSession) IsValid(ctx context.Context) bool  { return true }
func (s *passSession) Refresh(ctx context.Context) error { return nil }
func (s *passSession) ExpiresAt() time.Time              { return time.Time{} }
```

### Using Session Cache

Vaultmux provides `SessionCache` for persistent session storage:

```go
import "github.com/blackwell-systems/vaultmux"

// In your backend
func (b *Backend) Authenticate(ctx context.Context) (vaultmux.Session, error) {
    cache := vaultmux.NewSessionCache(b.sessionFile, 30*time.Minute)

    // Try to load cached session
    if cached, err := cache.Load(); err == nil && cached != nil {
        session := &mySession{token: cached.Token}
        if session.IsValid(ctx) {
            return session, nil
        }
    }

    // Authenticate fresh
    token, err := b.authenticate(ctx)
    if err != nil {
        return nil, err
    }

    // Cache the session
    _ = cache.Save(token, b.Name())

    return &mySession{token: token}, nil
}
```

---

## Implementation Steps

### Step 1: Create Backend Package

Create a new directory under `backends/`:

```bash
mkdir -p backends/yourbackend
touch backends/yourbackend/yourbackend.go
touch backends/yourbackend/yourbackend_test.go
```

### Step 2: Define Backend Struct

```go
package yourbackend

import (
    "context"
    "os/exec"

    "github.com/blackwell-systems/vaultmux"
)

// Backend implements vaultmux.Backend for YourVault.
type Backend struct {
    // Configuration
    apiURL      string
    sessionFile string
    prefix      string

    // Optional: cache frequently used data
    itemCache map[string]*vaultmux.Item
}

// New creates a new YourVault backend.
func New(options map[string]string, sessionFile string) (*Backend, error) {
    apiURL := options["api_url"]
    if apiURL == "" {
        apiURL = "https://api.yourvault.com"
    }

    prefix := options["prefix"]
    if prefix == "" {
        prefix = "dotfiles"
    }

    return &Backend{
        apiURL:      apiURL,
        sessionFile: sessionFile,
        prefix:      prefix,
        itemCache:   make(map[string]*vaultmux.Item),
    }, nil
}
```

### Step 3: Implement Core Methods

```go
func (b *Backend) Name() string {
    return "yourbackend"
}

func (b *Backend) Init(ctx context.Context) error {
    // Check CLI is installed
    if _, err := exec.LookPath("yourvault"); err != nil {
        return vaultmux.WrapError(b.Name(), "init", "",
            fmt.Errorf("CLI not installed: %w", err))
    }

    // Verify API reachability (optional)
    // Check config files exist

    return nil
}

func (b *Backend) Close() error {
    // Clean up resources
    b.itemCache = nil
    return nil
}
```

### Step 4: Implement Authentication

```go
func (b *Backend) IsAuthenticated(ctx context.Context) bool {
    // Quick check without prompting user
    cmd := exec.CommandContext(ctx, "yourvault", "status")
    out, err := cmd.Output()
    if err != nil {
        return false
    }

    // Parse output to check auth status
    return strings.Contains(string(out), "authenticated")
}

func (b *Backend) Authenticate(ctx context.Context) (vaultmux.Session, error) {
    // Try cached session first
    cache := vaultmux.NewSessionCache(b.sessionFile, 30*time.Minute)
    if cached, err := cache.Load(); err == nil && cached != nil {
        session := &yourSession{token: cached.Token, backend: b}
        if session.IsValid(ctx) {
            return session, nil
        }
    }

    // Perform fresh authentication
    cmd := exec.CommandContext(ctx, "yourvault", "login")
    cmd.Stdin = os.Stdin
    cmd.Stderr = os.Stderr

    out, err := cmd.Output()
    if err != nil {
        return nil, vaultmux.WrapError(b.Name(), "authenticate", "", err)
    }

    token := strings.TrimSpace(string(out))

    // Cache the session
    _ = cache.Save(token, b.Name())

    return &yourSession{
        token:   token,
        expires: time.Now().Add(30 * time.Minute),
        backend: b,
    }, nil
}
```

### Step 5: Implement Item Operations

```go
func (b *Backend) GetItem(ctx context.Context, name string, session vaultmux.Session) (*vaultmux.Item, error) {
    cmd := exec.CommandContext(ctx, "yourvault", "get", name,
        "--session", session.Token(), "--format", "json")

    out, err := cmd.Output()
    if err != nil {
        if strings.Contains(string(out), "not found") {
            return nil, vaultmux.ErrNotFound
        }
        return nil, vaultmux.WrapError(b.Name(), "get", name, err)
    }

    var result struct {
        ID    string `json:"id"`
        Name  string `json:"name"`
        Notes string `json:"notes"`
    }

    if err := json.Unmarshal(out, &result); err != nil {
        return nil, vaultmux.WrapError(b.Name(), "parse", name, err)
    }

    return &vaultmux.Item{
        ID:    result.ID,
        Name:  result.Name,
        Type:  vaultmux.ItemTypeSecureNote,
        Notes: result.Notes,
    }, nil
}

func (b *Backend) GetNotes(ctx context.Context, name string, session vaultmux.Session) (string, error) {
    item, err := b.GetItem(ctx, name, session)
    if err != nil {
        return "", err
    }
    return item.Notes, nil
}

func (b *Backend) ItemExists(ctx context.Context, name string, session vaultmux.Session) (bool, error) {
    _, err := b.GetItem(ctx, name, session)
    if err != nil {
        if errors.Is(err, vaultmux.ErrNotFound) {
            return false, nil
        }
        return false, err
    }
    return true, nil
}

func (b *Backend) ListItems(ctx context.Context, session vaultmux.Session) ([]*vaultmux.Item, error) {
    cmd := exec.CommandContext(ctx, "yourvault", "list",
        "--session", session.Token(), "--format", "json")

    out, err := cmd.Output()
    if err != nil {
        return nil, vaultmux.WrapError(b.Name(), "list", "", err)
    }

    var results []struct {
        ID   string `json:"id"`
        Name string `json:"name"`
    }

    if err := json.Unmarshal(out, &results); err != nil {
        return nil, vaultmux.WrapError(b.Name(), "parse", "", err)
    }

    items := make([]*vaultmux.Item, len(results))
    for i, r := range results {
        items[i] = &vaultmux.Item{
            ID:   r.ID,
            Name: r.Name,
            Type: vaultmux.ItemTypeSecureNote,
        }
    }

    return items, nil
}
```

### Step 6: Implement Mutations

```go
func (b *Backend) CreateItem(ctx context.Context, name, content string, session vaultmux.Session) error {
    exists, err := b.ItemExists(ctx, name, session)
    if err != nil {
        return err
    }
    if exists {
        return vaultmux.ErrAlreadyExists
    }

    cmd := exec.CommandContext(ctx, "yourvault", "create", name,
        "--session", session.Token(), "--notes", content)

    if err := cmd.Run(); err != nil {
        return vaultmux.WrapError(b.Name(), "create", name, err)
    }

    return nil
}

func (b *Backend) UpdateItem(ctx context.Context, name, content string, session vaultmux.Session) error {
    exists, err := b.ItemExists(ctx, name, session)
    if err != nil {
        return err
    }
    if !exists {
        return vaultmux.ErrNotFound
    }

    cmd := exec.CommandContext(ctx, "yourvault", "update", name,
        "--session", session.Token(), "--notes", content)

    if err := cmd.Run(); err != nil {
        return vaultmux.WrapError(b.Name(), "update", name, err)
    }

    return nil
}

func (b *Backend) DeleteItem(ctx context.Context, name string, session vaultmux.Session) error {
    exists, err := b.ItemExists(ctx, name, session)
    if err != nil {
        return err
    }
    if !exists {
        return vaultmux.ErrNotFound
    }

    cmd := exec.CommandContext(ctx, "yourvault", "delete", name,
        "--session", session.Token(), "--force")

    if err := cmd.Run(); err != nil {
        return vaultmux.WrapError(b.Name(), "delete", name, err)
    }

    return nil
}
```

### Step 7: Implement Sync (Optional)

```go
func (b *Backend) Sync(ctx context.Context, session vaultmux.Session) error {
    // If your backend supports syncing with a server:
    cmd := exec.CommandContext(ctx, "yourvault", "sync",
        "--session", session.Token())

    if err := cmd.Run(); err != nil {
        return vaultmux.WrapError(b.Name(), "sync", "", err)
    }

    return nil

    // If your backend doesn't support sync, just return nil:
    // return nil
}
```

### Step 8: Implement Session Type

```go
type yourSession struct {
    token   string
    expires time.Time
    backend *Backend
}

func (s *yourSession) Token() string {
    return s.token
}

func (s *yourSession) IsValid(ctx context.Context) bool {
    if time.Now().After(s.expires) {
        return false
    }

    // Verify with backend
    cmd := exec.CommandContext(ctx, "yourvault", "verify",
        "--session", s.token)
    return cmd.Run() == nil
}

func (s *yourSession) Refresh(ctx context.Context) error {
    newSession, err := s.backend.Authenticate(ctx)
    if err != nil {
        return err
    }
    s.token = newSession.Token()
    s.expires = newSession.ExpiresAt()
    return nil
}

func (s *yourSession) ExpiresAt() time.Time {
    return s.expires
}
```

### Step 9: Register Your Backend

Add a registration function using `init()`:

```go
func init() {
    vaultmux.RegisterBackend(vaultmux.BackendType("yourbackend"),
        func(cfg vaultmux.Config) (vaultmux.Backend, error) {
            return New(cfg.Options, cfg.SessionFile)
        })
}
```

---

## Testing Your Backend

### Unit Tests

Create comprehensive tests in `yourbackend_test.go`:

```go
package yourbackend

import (
    "context"
    "testing"

    "github.com/blackwell-systems/vaultmux"
)

func TestBackend_Name(t *testing.T) {
    backend, _ := New(nil, "")
    if got := backend.Name(); got != "yourbackend" {
        t.Errorf("Name() = %q, want %q", got, "yourbackend")
    }
}

func TestBackend_Init(t *testing.T) {
    backend, _ := New(nil, "")
    ctx := context.Background()

    // This will fail if CLI not installed - that's expected
    err := backend.Init(ctx)

    // Document that CLI must be installed for tests
    if err != nil {
        t.Skipf("CLI not installed: %v", err)
    }
}

func TestBackend_CRUD(t *testing.T) {
    backend, _ := New(nil, "")
    ctx := context.Background()

    // Authenticate
    session, err := backend.Authenticate(ctx)
    if err != nil {
        t.Skipf("Authentication failed: %v", err)
    }

    // Create
    err = backend.CreateItem(ctx, "test-item", "test-content", session)
    if err != nil {
        t.Fatalf("CreateItem() error = %v", err)
    }

    // Read
    notes, err := backend.GetNotes(ctx, "test-item", session)
    if err != nil {
        t.Fatalf("GetNotes() error = %v", err)
    }
    if notes != "test-content" {
        t.Errorf("GetNotes() = %q, want %q", notes, "test-content")
    }

    // Update
    err = backend.UpdateItem(ctx, "test-item", "updated-content", session)
    if err != nil {
        t.Fatalf("UpdateItem() error = %v", err)
    }

    // Delete
    err = backend.DeleteItem(ctx, "test-item", session)
    if err != nil {
        t.Fatalf("DeleteItem() error = %v", err)
    }

    // Verify deleted
    exists, _ := backend.ItemExists(ctx, "test-item", session)
    if exists {
        t.Error("ItemExists() = true after delete, want false")
    }
}
```

### Integration Tests

Test with the mock backend to ensure interface compliance:

```go
func TestBackend_InterfaceCompliance(t *testing.T) {
    var _ vaultmux.Backend = (*Backend)(nil)
}
```

---

## Registration and Usage

### Backend Registration

Add your backend to the registration system by importing it:

```go
// In your application
import (
    "github.com/blackwell-systems/vaultmux"
    _ "github.com/blackwell-systems/vaultmux/backends/yourbackend"
)

func main() {
    backend, err := vaultmux.New(vaultmux.Config{
        Backend:     vaultmux.BackendType("yourbackend"),
        Options:     map[string]string{"api_url": "https://custom.url"},
        SessionFile: "/path/to/.session",
        Prefix:      "myapp",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer backend.Close()

    // Use the backend
}
```

### Configuration Options

Document the options your backend accepts:

```go
// Supported options:
// - api_url: API endpoint URL (default: https://api.yourvault.com)
// - timeout: Request timeout in seconds (default: 30)
// - prefix: Item name prefix (default: "dotfiles")
// - custom_field: Backend-specific setting
```

---

## Best Practices

### Error Handling

Always use `vaultmux.WrapError()` for consistent error messages:

```go
if err := someOperation(); err != nil {
    return vaultmux.WrapError(b.Name(), "operation", itemName, err)
}
```

This produces: `[yourbackend:operation:itemName] original error message`

### Standard Errors

Return standard errors where appropriate:

```go
// Item not found
return nil, vaultmux.ErrNotFound

// Item already exists
return vaultmux.ErrAlreadyExists

// Not authenticated
return vaultmux.ErrNotAuthenticated
```

### Context Cancellation

Always respect context cancellation:

```go
func (b *Backend) GetItem(ctx context.Context, name string, session vaultmux.Session) (*vaultmux.Item, error) {
    // Check if context is already cancelled
    if err := ctx.Err(); err != nil {
        return nil, err
    }

    // Use CommandContext (not Command) for automatic cancellation
    cmd := exec.CommandContext(ctx, "yourvault", "get", name)

    // Context cancellation is handled automatically
    out, err := cmd.Output()
    // ...
}
```

### Session Validation

Always check session validity before operations:

```go
func (b *Backend) GetItem(ctx context.Context, name string, session vaultmux.Session) (*vaultmux.Item, error) {
    if !session.IsValid(ctx) {
        return nil, vaultmux.ErrNotAuthenticated
    }

    // Proceed with operation
}
```

### Item Naming

Apply prefix consistently:

```go
func (b *Backend) itemPath(name string) string {
    if b.prefix != "" {
        return filepath.Join(b.prefix, name)
    }
    return name
}

// Use in all operations
func (b *Backend) GetItem(ctx context.Context, name string, session vaultmux.Session) (*vaultmux.Item, error) {
    path := b.itemPath(name)
    cmd := exec.CommandContext(ctx, "yourvault", "get", path)
    // ...
}
```

### Location Management (Optional)

If your backend supports folders/vaults/collections:

```go
func (b *Backend) ListLocations(ctx context.Context, session vaultmux.Session) ([]string, error) {
    cmd := exec.CommandContext(ctx, "yourvault", "list-folders",
        "--session", session.Token())

    out, err := cmd.Output()
    if err != nil {
        return nil, vaultmux.WrapError(b.Name(), "list-locations", "", err)
    }

    // Parse and return location names
    var locations []string
    // ... parsing logic

    return locations, nil
}

func (b *Backend) LocationExists(ctx context.Context, name string, session vaultmux.Session) (bool, error) {
    locations, err := b.ListLocations(ctx, session)
    if err != nil {
        return false, err
    }

    for _, loc := range locations {
        if loc == name {
            return true, nil
        }
    }

    return false, nil
}

func (b *Backend) CreateLocation(ctx context.Context, name string, session vaultmux.Session) error {
    exists, err := b.LocationExists(ctx, name, session)
    if err != nil {
        return err
    }
    if exists {
        return vaultmux.ErrAlreadyExists
    }

    cmd := exec.CommandContext(ctx, "yourvault", "create-folder", name,
        "--session", session.Token())

    if err := cmd.Run(); err != nil {
        return vaultmux.WrapError(b.Name(), "create-location", name, err)
    }

    return nil
}

func (b *Backend) ListItemsInLocation(ctx context.Context, locType, locValue string, session vaultmux.Session) ([]*vaultmux.Item, error) {
    cmd := exec.CommandContext(ctx, "yourvault", "list",
        "--folder", locValue,
        "--session", session.Token(),
        "--format", "json")

    out, err := cmd.Output()
    if err != nil {
        return nil, vaultmux.WrapError(b.Name(), "list-in-location", locValue, err)
    }

    // Parse and return items
    var items []*vaultmux.Item
    // ... parsing logic

    return items, nil
}
```

---

## Complete Example

Here's a minimal but complete backend implementation:

```go
// backends/yourbackend/yourbackend.go

package yourbackend

import (
    "context"
    "encoding/json"
    "fmt"
    "os/exec"
    "strings"
    "time"

    "github.com/blackwell-systems/vaultmux"
)

// Backend implements vaultmux.Backend for YourVault.
type Backend struct {
    sessionFile string
    prefix      string
}

func New(options map[string]string, sessionFile string) (*Backend, error) {
    prefix := options["prefix"]
    if prefix == "" {
        prefix = "dotfiles"
    }

    return &Backend{
        sessionFile: sessionFile,
        prefix:      prefix,
    }, nil
}

func init() {
    vaultmux.RegisterBackend(vaultmux.BackendType("yourbackend"),
        func(cfg vaultmux.Config) (vaultmux.Backend, error) {
            return New(cfg.Options, cfg.SessionFile)
        })
}

func (b *Backend) Name() string { return "yourbackend" }

func (b *Backend) Init(ctx context.Context) error {
    if _, err := exec.LookPath("yourvault"); err != nil {
        return fmt.Errorf("yourvault CLI not installed: %w", err)
    }
    return nil
}

func (b *Backend) Close() error { return nil }

func (b *Backend) IsAuthenticated(ctx context.Context) bool {
    cmd := exec.CommandContext(ctx, "yourvault", "status")
    return cmd.Run() == nil
}

func (b *Backend) Authenticate(ctx context.Context) (vaultmux.Session, error) {
    // Check cached session
    cache := vaultmux.NewSessionCache(b.sessionFile, 30*time.Minute)
    if cached, err := cache.Load(); err == nil && cached != nil {
        session := &yourSession{token: cached.Token, backend: b}
        if session.IsValid(ctx) {
            return session, nil
        }
    }

    // Authenticate
    cmd := exec.CommandContext(ctx, "yourvault", "login", "--raw")
    out, err := cmd.Output()
    if err != nil {
        return nil, vaultmux.WrapError(b.Name(), "authenticate", "", err)
    }

    token := strings.TrimSpace(string(out))
    _ = cache.Save(token, b.Name())

    return &yourSession{
        token:   token,
        expires: time.Now().Add(30 * time.Minute),
        backend: b,
    }, nil
}

func (b *Backend) Sync(ctx context.Context, session vaultmux.Session) error {
    return nil // No-op if not supported
}

func (b *Backend) GetItem(ctx context.Context, name string, session vaultmux.Session) (*vaultmux.Item, error) {
    path := b.itemPath(name)
    cmd := exec.CommandContext(ctx, "yourvault", "get", path,
        "--session", session.Token(), "--json")

    out, err := cmd.Output()
    if err != nil {
        if strings.Contains(string(out), "not found") {
            return nil, vaultmux.ErrNotFound
        }
        return nil, vaultmux.WrapError(b.Name(), "get", name, err)
    }

    var item struct {
        ID    string `json:"id"`
        Name  string `json:"name"`
        Notes string `json:"notes"`
    }

    if err := json.Unmarshal(out, &item); err != nil {
        return nil, vaultmux.WrapError(b.Name(), "parse", name, err)
    }

    return &vaultmux.Item{
        ID:    item.ID,
        Name:  item.Name,
        Type:  vaultmux.ItemTypeSecureNote,
        Notes: item.Notes,
    }, nil
}

func (b *Backend) GetNotes(ctx context.Context, name string, session vaultmux.Session) (string, error) {
    item, err := b.GetItem(ctx, name, session)
    if err != nil {
        return "", err
    }
    return item.Notes, nil
}

func (b *Backend) ItemExists(ctx context.Context, name string, session vaultmux.Session) (bool, error) {
    _, err := b.GetItem(ctx, name, session)
    if err != nil {
        if errors.Is(err, vaultmux.ErrNotFound) {
            return false, nil
        }
        return false, err
    }
    return true, nil
}

func (b *Backend) ListItems(ctx context.Context, session vaultmux.Session) ([]*vaultmux.Item, error) {
    cmd := exec.CommandContext(ctx, "yourvault", "list",
        "--prefix", b.prefix,
        "--session", session.Token(),
        "--json")

    out, err := cmd.Output()
    if err != nil {
        return nil, vaultmux.WrapError(b.Name(), "list", "", err)
    }

    var items []*vaultmux.Item
    if err := json.Unmarshal(out, &items); err != nil {
        return nil, vaultmux.WrapError(b.Name(), "parse", "", err)
    }

    return items, nil
}

func (b *Backend) CreateItem(ctx context.Context, name, content string, session vaultmux.Session) error {
    exists, err := b.ItemExists(ctx, name, session)
    if err != nil {
        return err
    }
    if exists {
        return vaultmux.ErrAlreadyExists
    }

    path := b.itemPath(name)
    cmd := exec.CommandContext(ctx, "yourvault", "create", path,
        "--notes", content,
        "--session", session.Token())

    if err := cmd.Run(); err != nil {
        return vaultmux.WrapError(b.Name(), "create", name, err)
    }

    return nil
}

func (b *Backend) UpdateItem(ctx context.Context, name, content string, session vaultmux.Session) error {
    exists, err := b.ItemExists(ctx, name, session)
    if err != nil {
        return err
    }
    if !exists {
        return vaultmux.ErrNotFound
    }

    path := b.itemPath(name)
    cmd := exec.CommandContext(ctx, "yourvault", "update", path,
        "--notes", content,
        "--session", session.Token())

    if err := cmd.Run(); err != nil {
        return vaultmux.WrapError(b.Name(), "update", name, err)
    }

    return nil
}

func (b *Backend) DeleteItem(ctx context.Context, name string, session vaultmux.Session) error {
    exists, err := b.ItemExists(ctx, name, session)
    if err != nil {
        return err
    }
    if !exists {
        return vaultmux.ErrNotFound
    }

    path := b.itemPath(name)
    cmd := exec.CommandContext(ctx, "yourvault", "delete", path,
        "--session", session.Token(),
        "--force")

    if err := cmd.Run(); err != nil {
        return vaultmux.WrapError(b.Name(), "delete", name, err)
    }

    return nil
}

func (b *Backend) itemPath(name string) string {
    if b.prefix != "" {
        return b.prefix + "/" + name
    }
    return name
}

// Location management stubs (implement if supported)
func (b *Backend) ListLocations(ctx context.Context, session vaultmux.Session) ([]string, error) {
    return nil, fmt.Errorf("location management not supported")
}

func (b *Backend) LocationExists(ctx context.Context, name string, session vaultmux.Session) (bool, error) {
    return false, fmt.Errorf("location management not supported")
}

func (b *Backend) CreateLocation(ctx context.Context, name string, session vaultmux.Session) error {
    return fmt.Errorf("location management not supported")
}

func (b *Backend) ListItemsInLocation(ctx context.Context, locType, locValue string, session vaultmux.Session) ([]*vaultmux.Item, error) {
    return nil, fmt.Errorf("location management not supported")
}

// Session implementation
type yourSession struct {
    token   string
    expires time.Time
    backend *Backend
}

func (s *yourSession) Token() string { return s.token }

func (s *yourSession) IsValid(ctx context.Context) bool {
    if time.Now().After(s.expires) {
        return false
    }
    cmd := exec.CommandContext(ctx, "yourvault", "verify", "--session", s.token)
    return cmd.Run() == nil
}

func (s *yourSession) Refresh(ctx context.Context) error {
    newSession, err := s.backend.Authenticate(ctx)
    if err != nil {
        return err
    }
    s.token = newSession.Token()
    s.expires = newSession.ExpiresAt()
    return nil
}

func (s *yourSession) ExpiresAt() time.Time {
    return s.expires
}
```

---

## Testing Checklist

Before submitting your backend:

- [ ] All `Backend` interface methods implemented
- [ ] Session type implements `Session` interface
- [ ] Registration function in `init()`
- [ ] Unit tests for all CRUD operations
- [ ] Error handling uses `vaultmux.WrapError()`
- [ ] Standard errors (`ErrNotFound`, `ErrAlreadyExists`) returned appropriately
- [ ] Context cancellation respected
- [ ] Session caching works (if applicable)
- [ ] Documentation includes all configuration options
- [ ] CLI tool installation checked in `Init()`
- [ ] No panics (all errors returned as values)

---

## Reference Implementations

Study these backends for implementation patterns:

### **pass** (`backends/pass/pass.go`)
- Best for: File-based, simple backends
- Features: No session management, direct filesystem access
- Pattern: GPG agent handles authentication

### **bitwarden** (`backends/bitwarden/bitwarden.go`)
- Best for: CLI-based, token session backends
- Features: Session caching, JSON parsing, status checking
- Pattern: Token-based authentication with expiry

### **onepassword** (`backends/onepassword/onepassword.go`)
- Best for: API-based backends with complex authentication
- Features: Account management, vault selection, service accounts
- Pattern: Token-based with location support

### **wincred** (`backends/wincred/wincred_windows.go`)
- Best for: Platform-specific backends, OS-integrated auth
- Features: No session management, build tags for cross-platform, PowerShell interop
- Pattern: OS-level authentication, no tokens
- Platform: Windows only (graceful error on Unix via build tags)

---

## Contributing Your Backend

Once your backend is ready:

1. **Test thoroughly** with real vault operations
2. **Add documentation** to README.md listing your backend
3. **Update examples** showing your backend usage
4. **Submit PR** to vaultmux repository
5. **Maintain** your backend implementation

Your backend will be available to all vaultmux users via:

```go
import _ "github.com/blackwell-systems/vaultmux/backends/yourbackend"
```

---

## Support and Questions

- **Issues**: https://github.com/blackwell-systems/vaultmux/issues
- **Discussions**: https://github.com/blackwell-systems/vaultmux/discussions
- **Examples**: See `backends/` directory for reference implementations

---

**Version**: 1.0
**Last Updated**: 2025-12-07
**Vaultmux Version**: v0.1.0
