// Package onepassword implements the vaultmux.Backend interface for 1Password CLI.
package onepassword

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/blackwell-systems/vaultmux"
)

func init() {
	vaultmux.RegisterBackend(vaultmux.BackendOnePassword, func(cfg vaultmux.Config) (vaultmux.Backend, error) {
		return New(cfg.Options, cfg.SessionFile)
	})
}

// statusCache caches the result of IsAuthenticated checks to reduce subprocess overhead.
type statusCache struct {
	authenticated bool
	timestamp     time.Time
	mu            sync.RWMutex
}

// get returns the cached status if still valid (within TTL).
func (s *statusCache) get(ttl time.Duration) (bool, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if time.Since(s.timestamp) < ttl {
		return s.authenticated, true // cached result is valid
	}
	return false, false // cache expired
}

// set updates the cached status with current timestamp.
func (s *statusCache) set(authenticated bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.authenticated = authenticated
	s.timestamp = time.Now()
}

// Backend implements vaultmux.Backend for 1Password CLI (op).
type Backend struct {
	sessionFile string
	cache       *vaultmux.SessionCache
	statusCache statusCache // Caches IsAuthenticated results
}

// New creates a new 1Password backend.
func New(opts map[string]string, sessionFile string) (*Backend, error) {
	if sessionFile == "" {
		home, _ := os.UserHomeDir()
		sessionFile = filepath.Join(home, ".config", "vaultmux", ".op-session")
	}

	return &Backend{
		sessionFile: sessionFile,
		cache:       vaultmux.NewSessionCache(sessionFile, 30*time.Minute),
	}, nil
}

// Name returns the backend name.
func (b *Backend) Name() string { return "1password" }

// Init checks if the 1Password CLI is installed.
func (b *Backend) Init(ctx context.Context) error {
	if _, err := exec.LookPath("op"); err != nil {
		return vaultmux.ErrBackendNotInstalled
	}
	return nil
}

// Close is a no-op for 1Password.
func (b *Backend) Close() error { return nil }

// IsAuthenticated checks if there's a valid session.
// Results are cached for 5 seconds to reduce subprocess overhead.
func (b *Backend) IsAuthenticated(ctx context.Context) bool {
	// Check cache first (5 second TTL)
	if result, valid := b.statusCache.get(5 * time.Second); valid {
		return result
	}

	// Try loading cached session
	cached, err := b.cache.Load()
	if err != nil || cached == nil {
		b.statusCache.set(false)
		return false
	}

	// Verify with op whoami
	cmd := exec.CommandContext(ctx, "op", "whoami", "--format", "json")
	cmd.Env = append(os.Environ(), fmt.Sprintf("OP_SESSION_%s=%s", "my", cached.Token))
	authenticated := cmd.Run() == nil

	// Cache the result
	b.statusCache.set(authenticated)
	return authenticated
}

// Authenticate signs in to 1Password and returns a session.
func (b *Backend) Authenticate(ctx context.Context) (vaultmux.Session, error) {
	// Try cached session first
	if cached, err := b.cache.Load(); err == nil && cached != nil {
		sess := &opSession{token: cached.Token, backend: b}
		if sess.IsValid(ctx) {
			return sess, nil
		}
	}

	// Run: op signin --raw
	cmd := exec.CommandContext(ctx, "op", "signin", "--raw")
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()
	if err != nil {
		return nil, vaultmux.WrapError("1password", "authenticate", "", err)
	}

	token := strings.TrimSpace(string(out))

	// Cache the session
	_ = b.cache.Save(token, "1password")

	// Update status cache since we just authenticated
	b.statusCache.set(true)

	return &opSession{
		token:   token,
		backend: b,
		expires: time.Now().Add(30 * time.Minute),
	}, nil
}

// Sync is a no-op for 1Password (syncs automatically).
func (b *Backend) Sync(ctx context.Context, session vaultmux.Session) error {
	return nil // 1Password syncs automatically
}

// GetItem retrieves a vault item by name.
func (b *Backend) GetItem(ctx context.Context, name string, session vaultmux.Session) (*vaultmux.Item, error) {
	if err := vaultmux.ValidateItemName(name); err != nil {
		return nil, vaultmux.WrapError("1password", "get", name, err)
	}

	cmd := exec.CommandContext(ctx, "op", "item", "get", name, "--format", "json")
	cmd.Env = b.sessionEnv(session)

	out, err := cmd.Output()
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, vaultmux.ErrNotFound
		}
		return nil, vaultmux.WrapError("1password", "get", name, err)
	}

	var opItem struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		Vault struct {
			Name string `json:"name"`
		} `json:"vault"`
		Fields []struct {
			ID    string `json:"id"`
			Type  string `json:"type"`
			Label string `json:"label"`
			Value string `json:"value"`
		} `json:"fields"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}

	if err := json.Unmarshal(out, &opItem); err != nil {
		return nil, vaultmux.WrapError("1password", "parse", name, err)
	}

	// Extract notes field
	var notes string
	for _, field := range opItem.Fields {
		if field.Label == "notesPlain" || field.Type == "TEXT" {
			notes = field.Value
			break
		}
	}

	return &vaultmux.Item{
		ID:       opItem.ID,
		Name:     opItem.Title,
		Type:     vaultmux.ItemTypeSecureNote,
		Notes:    notes,
		Location: opItem.Vault.Name,
		Created:  opItem.CreatedAt,
		Modified: opItem.UpdatedAt,
	}, nil
}

// GetNotes retrieves just the notes field of an item.
func (b *Backend) GetNotes(ctx context.Context, name string, session vaultmux.Session) (string, error) {
	item, err := b.GetItem(ctx, name, session)
	if err != nil {
		return "", err
	}
	return item.Notes, nil
}

// ItemExists checks if an item exists.
func (b *Backend) ItemExists(ctx context.Context, name string, session vaultmux.Session) (bool, error) {
	_, err := b.GetItem(ctx, name, session)
	if err == vaultmux.ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// ListItems lists all items in the vault.
func (b *Backend) ListItems(ctx context.Context, session vaultmux.Session) ([]*vaultmux.Item, error) {
	cmd := exec.CommandContext(ctx, "op", "item", "list", "--format", "json")
	cmd.Env = b.sessionEnv(session)

	out, err := cmd.Output()
	if err != nil {
		return nil, vaultmux.WrapError("1password", "list", "", err)
	}

	var opItems []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
		Vault struct {
			Name string `json:"name"`
		} `json:"vault"`
	}

	if err := json.Unmarshal(out, &opItems); err != nil {
		return nil, vaultmux.WrapError("1password", "parse-list", "", err)
	}

	items := make([]*vaultmux.Item, len(opItems))
	for i, opItem := range opItems {
		items[i] = &vaultmux.Item{
			ID:       opItem.ID,
			Name:     opItem.Title,
			Type:     vaultmux.ItemTypeSecureNote,
			Location: opItem.Vault.Name,
		}
	}

	return items, nil
}

// CreateItem creates a new secure note.
func (b *Backend) CreateItem(ctx context.Context, name, content string, session vaultmux.Session) error {
	if err := vaultmux.ValidateItemName(name); err != nil {
		return vaultmux.WrapError("1password", "create", name, err)
	}

	cmd := exec.CommandContext(ctx, "op", "item", "create",
		"--category", "Secure Note",
		"--title", name,
		fmt.Sprintf("notesPlain=%s", content))
	cmd.Env = b.sessionEnv(session)

	if err := cmd.Run(); err != nil {
		return vaultmux.WrapError("1password", "create", name, err)
	}

	return nil
}

// UpdateItem updates an existing item's notes.
func (b *Backend) UpdateItem(ctx context.Context, name, content string, session vaultmux.Session) error {
	if err := vaultmux.ValidateItemName(name); err != nil {
		return vaultmux.WrapError("1password", "update", name, err)
	}

	cmd := exec.CommandContext(ctx, "op", "item", "edit", name,
		fmt.Sprintf("notesPlain=%s", content))
	cmd.Env = b.sessionEnv(session)

	if err := cmd.Run(); err != nil {
		return vaultmux.WrapError("1password", "update", name, err)
	}

	return nil
}

// DeleteItem deletes an item.
func (b *Backend) DeleteItem(ctx context.Context, name string, session vaultmux.Session) error {
	if err := vaultmux.ValidateItemName(name); err != nil {
		return vaultmux.WrapError("1password", "delete", name, err)
	}

	cmd := exec.CommandContext(ctx, "op", "item", "delete", name)
	cmd.Env = b.sessionEnv(session)

	if err := cmd.Run(); err != nil {
		return vaultmux.WrapError("1password", "delete", name, err)
	}

	return nil
}

// ListLocations lists vaults.
func (b *Backend) ListLocations(ctx context.Context, session vaultmux.Session) ([]string, error) {
	cmd := exec.CommandContext(ctx, "op", "vault", "list", "--format", "json")
	cmd.Env = b.sessionEnv(session)

	out, err := cmd.Output()
	if err != nil {
		return nil, vaultmux.WrapError("1password", "list-vaults", "", err)
	}

	var vaults []struct {
		Name string `json:"name"`
	}

	if err := json.Unmarshal(out, &vaults); err != nil {
		return nil, vaultmux.WrapError("1password", "parse-vaults", "", err)
	}

	locations := make([]string, len(vaults))
	for i, vault := range vaults {
		locations[i] = vault.Name
	}

	return locations, nil
}

// LocationExists checks if a vault exists.
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

// CreateLocation creates a new vault.
func (b *Backend) CreateLocation(ctx context.Context, name string, session vaultmux.Session) error {
	if err := vaultmux.ValidateLocationName(name); err != nil {
		return vaultmux.WrapError("1password", "create-vault", name, err)
	}

	cmd := exec.CommandContext(ctx, "op", "vault", "create", name)
	cmd.Env = b.sessionEnv(session)

	if err := cmd.Run(); err != nil {
		return vaultmux.WrapError("1password", "create-vault", name, err)
	}

	return nil
}

// ListItemsInLocation lists items in a specific vault.
func (b *Backend) ListItemsInLocation(ctx context.Context, locType, locValue string, session vaultmux.Session) ([]*vaultmux.Item, error) {
	cmd := exec.CommandContext(ctx, "op", "item", "list", "--vault", locValue, "--format", "json")
	cmd.Env = b.sessionEnv(session)

	out, err := cmd.Output()
	if err != nil {
		return nil, vaultmux.WrapError("1password", "list-items-in-vault", locValue, err)
	}

	var opItems []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}

	if err := json.Unmarshal(out, &opItems); err != nil {
		return nil, vaultmux.WrapError("1password", "parse-items", locValue, err)
	}

	items := make([]*vaultmux.Item, len(opItems))
	for i, opItem := range opItems {
		items[i] = &vaultmux.Item{
			ID:       opItem.ID,
			Name:     opItem.Title,
			Type:     vaultmux.ItemTypeSecureNote,
			Location: locValue,
		}
	}

	return items, nil
}

// sessionEnv returns environment with session token set.
func (b *Backend) sessionEnv(session vaultmux.Session) []string {
	env := os.Environ()
	// 1Password uses OP_SESSION_<account> format, we'll use "my" as default
	env = append(env, fmt.Sprintf("OP_SESSION_my=%s", session.Token()))
	return env
}

// opSession implements vaultmux.Session for 1Password.
type opSession struct {
	token   string
	backend *Backend
	expires time.Time
}

func (s *opSession) Token() string { return s.token }

func (s *opSession) IsValid(ctx context.Context) bool {
	if time.Now().After(s.expires) {
		return false
	}
	cmd := exec.CommandContext(ctx, "op", "whoami", "--format", "json")
	cmd.Env = s.backend.sessionEnv(s)
	return cmd.Run() == nil
}

func (s *opSession) Refresh(ctx context.Context) error {
	// Re-authenticate
	newSession, err := s.backend.Authenticate(ctx)
	if err != nil {
		return err
	}
	s.token = newSession.Token()
	s.expires = newSession.ExpiresAt()
	return nil
}

func (s *opSession) ExpiresAt() time.Time {
	return s.expires
}
