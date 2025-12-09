// Package bitwarden implements the vaultmux.Backend interface for Bitwarden CLI.
package bitwarden

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
	vaultmux.RegisterBackend(vaultmux.BackendBitwarden, func(cfg vaultmux.Config) (vaultmux.Backend, error) {
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

// Backend implements vaultmux.Backend for Bitwarden CLI.
type Backend struct {
	sessionFile string
	cache       *vaultmux.SessionCache
	statusCache statusCache // Caches IsAuthenticated results
}

// New creates a new Bitwarden backend.
func New(opts map[string]string, sessionFile string) (*Backend, error) {
	if sessionFile == "" {
		home, _ := os.UserHomeDir()
		sessionFile = filepath.Join(home, ".config", "vaultmux", ".bw-session")
	}

	return &Backend{
		sessionFile: sessionFile,
		cache:       vaultmux.NewSessionCache(sessionFile, 30*time.Minute),
	}, nil
}

// Name returns the backend name.
func (b *Backend) Name() string { return "bitwarden" }

// Init checks if the Bitwarden CLI is installed.
func (b *Backend) Init(ctx context.Context) error {
	if _, err := exec.LookPath("bw"); err != nil {
		return vaultmux.ErrBackendNotInstalled
	}
	return nil
}

// Close is a no-op for Bitwarden.
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

	// Verify with bw status
	cmd := exec.CommandContext(ctx, "bw", "unlock", "--check")
	cmd.Env = append(os.Environ(), "BW_SESSION="+cached.Token)
	authenticated := cmd.Run() == nil

	// Cache the result
	b.statusCache.set(authenticated)
	return authenticated
}

// Authenticate unlocks the Bitwarden vault and returns a session.
func (b *Backend) Authenticate(ctx context.Context) (vaultmux.Session, error) {
	// Try cached session first
	if cached, err := b.cache.Load(); err == nil && cached != nil {
		sess := &bwSession{token: cached.Token, backend: b}
		if sess.IsValid(ctx) {
			return sess, nil
		}
	}

	// Check login status
	cmd := exec.CommandContext(ctx, "bw", "status")
	out, _ := cmd.Output()

	var status struct {
		Status string `json:"status"`
	}
	_ = json.Unmarshal(out, &status)

	if status.Status == "unauthenticated" {
		return nil, fmt.Errorf("not logged in to Bitwarden - run: bw login")
	}

	// Unlock and get session
	cmd = exec.CommandContext(ctx, "bw", "unlock", "--raw")
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()
	if err != nil {
		return nil, vaultmux.WrapError("bitwarden", "authenticate", "", err)
	}

	token := strings.TrimSpace(string(out))

	// Cache the session
	_ = b.cache.Save(token, "bitwarden")

	// Update status cache since we just authenticated
	b.statusCache.set(true)

	return &bwSession{token: token, backend: b}, nil
}

// Sync synchronizes the vault with the server.
func (b *Backend) Sync(ctx context.Context, session vaultmux.Session) error {
	cmd := exec.CommandContext(ctx, "bw", "sync")
	cmd.Env = append(os.Environ(), "BW_SESSION="+session.Token())
	if err := cmd.Run(); err != nil {
		return vaultmux.WrapError("bitwarden", "sync", "", err)
	}
	return nil
}

// GetItem retrieves a vault item by name.
func (b *Backend) GetItem(ctx context.Context, name string, session vaultmux.Session) (*vaultmux.Item, error) {
	if err := vaultmux.ValidateItemName(name); err != nil {
		return nil, vaultmux.WrapError("bitwarden", "get", name, err)
	}

	cmd := exec.CommandContext(ctx, "bw", "get", "item", name)
	cmd.Env = append(os.Environ(), "BW_SESSION="+session.Token())
	out, err := cmd.Output()
	if err != nil {
		if strings.Contains(string(out), "Not found") {
			return nil, vaultmux.ErrNotFound
		}
		return nil, vaultmux.WrapError("bitwarden", "get", name, err)
	}

	var bwItem struct {
		ID       string    `json:"id"`
		Name     string    `json:"name"`
		Type     int       `json:"type"`
		Notes    string    `json:"notes"`
		FolderID string    `json:"folderId"`
		Created  time.Time `json:"revisionDate"`
	}

	if err := json.Unmarshal(out, &bwItem); err != nil {
		return nil, vaultmux.WrapError("bitwarden", "parse", name, err)
	}

	// Map Bitwarden type to vaultmux ItemType
	itemType := vaultmux.ItemTypeSecureNote
	if bwItem.Type >= 0 && bwItem.Type <= 4 {
		itemType = vaultmux.ItemType(bwItem.Type)
	}

	return &vaultmux.Item{
		ID:       bwItem.ID,
		Name:     bwItem.Name,
		Type:     itemType,
		Notes:    bwItem.Notes,
		Location: bwItem.FolderID,
		Created:  bwItem.Created,
		Modified: bwItem.Created,
	}, nil
}

// GetNotes retrieves just the notes field of an item.
func (b *Backend) GetNotes(ctx context.Context, name string, session vaultmux.Session) (string, error) {
	item, err := b.GetItem(ctx, name, session)
	if err != nil {
		return "", err
	}
	if item == nil {
		return "", vaultmux.ErrNotFound
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
	cmd := exec.CommandContext(ctx, "bw", "list", "items")
	cmd.Env = append(os.Environ(), "BW_SESSION="+session.Token())
	out, err := cmd.Output()
	if err != nil {
		return nil, vaultmux.WrapError("bitwarden", "list", "", err)
	}

	var bwItems []struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Type  int    `json:"type"`
		Notes string `json:"notes"`
	}

	if err := json.Unmarshal(out, &bwItems); err != nil {
		return nil, vaultmux.WrapError("bitwarden", "parse-list", "", err)
	}

	items := make([]*vaultmux.Item, len(bwItems))
	for i, bwItem := range bwItems {
		itemType := vaultmux.ItemTypeSecureNote
		if bwItem.Type >= 0 && bwItem.Type <= 4 {
			itemType = vaultmux.ItemType(bwItem.Type)
		}
		items[i] = &vaultmux.Item{
			ID:    bwItem.ID,
			Name:  bwItem.Name,
			Type:  itemType,
			Notes: bwItem.Notes,
		}
	}

	return items, nil
}

// CreateItem creates a new secure note.
func (b *Backend) CreateItem(ctx context.Context, name, content string, session vaultmux.Session) error {
	if err := vaultmux.ValidateItemName(name); err != nil {
		return vaultmux.WrapError("bitwarden", "create", name, err)
	}

	// Create JSON template
	template := map[string]interface{}{
		"type":  2, // Secure note
		"name":  name,
		"notes": content,
		"secureNote": map[string]interface{}{
			"type": 0, // Generic
		},
	}

	jsonData, _ := json.Marshal(template)

	// Encode as base64 for bw
	cmd := exec.CommandContext(ctx, "bw", "encode")
	cmd.Stdin = strings.NewReader(string(jsonData))
	encoded, err := cmd.Output()
	if err != nil {
		return vaultmux.WrapError("bitwarden", "encode", name, err)
	}

	// Create item
	cmd = exec.CommandContext(ctx, "bw", "create", "item", strings.TrimSpace(string(encoded)))
	cmd.Env = append(os.Environ(), "BW_SESSION="+session.Token())
	if err := cmd.Run(); err != nil {
		return vaultmux.WrapError("bitwarden", "create", name, err)
	}

	return nil
}

// UpdateItem updates an existing item's notes.
func (b *Backend) UpdateItem(ctx context.Context, name, content string, session vaultmux.Session) error {
	// Get existing item
	item, err := b.GetItem(ctx, name, session)
	if err != nil {
		return err
	}

	// Update notes field
	template := map[string]interface{}{
		"type":  item.Type,
		"name":  item.Name,
		"notes": content,
	}

	jsonData, _ := json.Marshal(template)

	// Encode
	cmd := exec.CommandContext(ctx, "bw", "encode")
	cmd.Stdin = strings.NewReader(string(jsonData))
	encoded, err := cmd.Output()
	if err != nil {
		return vaultmux.WrapError("bitwarden", "encode", name, err)
	}

	// Edit item
	cmd = exec.CommandContext(ctx, "bw", "edit", "item", item.ID, strings.TrimSpace(string(encoded)))
	cmd.Env = append(os.Environ(), "BW_SESSION="+session.Token())
	if err := cmd.Run(); err != nil {
		return vaultmux.WrapError("bitwarden", "update", name, err)
	}

	return nil
}

// DeleteItem deletes an item.
func (b *Backend) DeleteItem(ctx context.Context, name string, session vaultmux.Session) error {
	// Get item to find ID
	item, err := b.GetItem(ctx, name, session)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, "bw", "delete", "item", item.ID)
	cmd.Env = append(os.Environ(), "BW_SESSION="+session.Token())
	if err := cmd.Run(); err != nil {
		return vaultmux.WrapError("bitwarden", "delete", name, err)
	}

	return nil
}

// ListLocations lists folders.
func (b *Backend) ListLocations(ctx context.Context, session vaultmux.Session) ([]string, error) {
	cmd := exec.CommandContext(ctx, "bw", "list", "folders")
	cmd.Env = append(os.Environ(), "BW_SESSION="+session.Token())
	out, err := cmd.Output()
	if err != nil {
		return nil, vaultmux.WrapError("bitwarden", "list-folders", "", err)
	}

	var folders []struct {
		Name string `json:"name"`
	}

	if err := json.Unmarshal(out, &folders); err != nil {
		return nil, vaultmux.WrapError("bitwarden", "parse-folders", "", err)
	}

	locations := make([]string, len(folders))
	for i, folder := range folders {
		locations[i] = folder.Name
	}

	return locations, nil
}

// LocationExists checks if a folder exists.
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

// CreateLocation creates a new folder.
func (b *Backend) CreateLocation(ctx context.Context, name string, session vaultmux.Session) error {
	if err := vaultmux.ValidateLocationName(name); err != nil {
		return vaultmux.WrapError("bitwarden", "create-folder", name, err)
	}

	template := map[string]interface{}{
		"name": name,
	}
	jsonData, _ := json.Marshal(template)

	cmd := exec.CommandContext(ctx, "bw", "encode")
	cmd.Stdin = strings.NewReader(string(jsonData))
	encoded, err := cmd.Output()
	if err != nil {
		return vaultmux.WrapError("bitwarden", "encode-folder", name, err)
	}

	cmd = exec.CommandContext(ctx, "bw", "create", "folder", strings.TrimSpace(string(encoded)))
	cmd.Env = append(os.Environ(), "BW_SESSION="+session.Token())
	if err := cmd.Run(); err != nil {
		return vaultmux.WrapError("bitwarden", "create-folder", name, err)
	}

	return nil
}

// ListItemsInLocation lists items in a specific folder.
func (b *Backend) ListItemsInLocation(ctx context.Context, locType, locValue string, session vaultmux.Session) ([]*vaultmux.Item, error) {
	// Get all items and filter
	allItems, err := b.ListItems(ctx, session)
	if err != nil {
		return nil, err
	}

	var items []*vaultmux.Item
	for _, item := range allItems {
		if item.Location == locValue {
			items = append(items, item)
		}
	}

	return items, nil
}

// bwSession implements vaultmux.Session for Bitwarden.
type bwSession struct {
	token   string
	backend *Backend
}

func (s *bwSession) Token() string { return s.token }

func (s *bwSession) IsValid(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "bw", "unlock", "--check")
	cmd.Env = append(os.Environ(), "BW_SESSION="+s.token)
	return cmd.Run() == nil
}

func (s *bwSession) Refresh(ctx context.Context) error {
	// Re-authenticate
	newSession, err := s.backend.Authenticate(ctx)
	if err != nil {
		return err
	}
	s.token = newSession.Token()
	return nil
}

func (s *bwSession) ExpiresAt() time.Time {
	// Bitwarden sessions don't have a fixed expiry - they expire when locked
	return time.Time{}
}
