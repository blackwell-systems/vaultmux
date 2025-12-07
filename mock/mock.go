// Package mock provides an in-memory mock implementation of vaultmux.Backend for testing.
package mock

import (
	"context"
	"sync"
	"time"

	"github.com/blackwell-systems/vaultmux"
)

// Backend is an in-memory mock for testing.
type Backend struct {
	items     map[string]*vaultmux.Item
	locations map[string]bool
	mu        sync.RWMutex

	// Behavior control for testing
	AuthError   error
	GetError    error
	CreateError error
	UpdateError error
	DeleteError error
	SyncError   error
}

// New creates a new mock backend.
func New() *Backend {
	return &Backend{
		items:     make(map[string]*vaultmux.Item),
		locations: make(map[string]bool),
	}
}

// Name returns the backend name.
func (b *Backend) Name() string { return "mock" }

// Init is a no-op for mock.
func (b *Backend) Init(ctx context.Context) error { return nil }

// Close is a no-op for mock.
func (b *Backend) Close() error { return nil }

// IsAuthenticated returns true unless AuthError is set.
func (b *Backend) IsAuthenticated(ctx context.Context) bool {
	return b.AuthError == nil
}

// Authenticate returns a mock session or AuthError.
func (b *Backend) Authenticate(ctx context.Context) (vaultmux.Session, error) {
	if b.AuthError != nil {
		return nil, b.AuthError
	}
	return &mockSession{}, nil
}

// Sync returns SyncError if set.
func (b *Backend) Sync(ctx context.Context, session vaultmux.Session) error {
	return b.SyncError
}

// GetItem retrieves an item from the in-memory store.
func (b *Backend) GetItem(ctx context.Context, name string, _ vaultmux.Session) (*vaultmux.Item, error) {
	if b.GetError != nil {
		return nil, b.GetError
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	item, ok := b.items[name]
	if !ok {
		return nil, vaultmux.ErrNotFound
	}

	// Return a copy to avoid race conditions
	itemCopy := *item
	return &itemCopy, nil
}

// GetNotes retrieves just the notes field.
func (b *Backend) GetNotes(ctx context.Context, name string, session vaultmux.Session) (string, error) {
	item, err := b.GetItem(ctx, name, session)
	if err != nil {
		return "", err
	}
	return item.Notes, nil
}

// ItemExists checks if an item exists.
func (b *Backend) ItemExists(ctx context.Context, name string, _ vaultmux.Session) (bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	_, ok := b.items[name]
	return ok, nil
}

// ListItems returns all items.
func (b *Backend) ListItems(ctx context.Context, _ vaultmux.Session) ([]*vaultmux.Item, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	items := make([]*vaultmux.Item, 0, len(b.items))
	for _, item := range b.items {
		itemCopy := *item
		items = append(items, &itemCopy)
	}

	return items, nil
}

// CreateItem creates a new item.
func (b *Backend) CreateItem(ctx context.Context, name, content string, _ vaultmux.Session) error {
	if b.CreateError != nil {
		return b.CreateError
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.items[name]; exists {
		return vaultmux.ErrAlreadyExists
	}

	now := time.Now()
	b.items[name] = &vaultmux.Item{
		ID:       name, // Use name as ID for simplicity
		Name:     name,
		Type:     vaultmux.ItemTypeSecureNote,
		Notes:    content,
		Created:  now,
		Modified: now,
	}

	return nil
}

// UpdateItem updates an existing item.
func (b *Backend) UpdateItem(ctx context.Context, name, content string, _ vaultmux.Session) error {
	if b.UpdateError != nil {
		return b.UpdateError
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	item, ok := b.items[name]
	if !ok {
		return vaultmux.ErrNotFound
	}

	item.Notes = content
	item.Modified = time.Now()

	return nil
}

// DeleteItem deletes an item.
func (b *Backend) DeleteItem(ctx context.Context, name string, _ vaultmux.Session) error {
	if b.DeleteError != nil {
		return b.DeleteError
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.items[name]; !ok {
		return vaultmux.ErrNotFound
	}

	delete(b.items, name)
	return nil
}

// ListLocations lists all locations.
func (b *Backend) ListLocations(ctx context.Context, _ vaultmux.Session) ([]string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	locations := make([]string, 0, len(b.locations))
	for loc := range b.locations {
		locations = append(locations, loc)
	}

	return locations, nil
}

// LocationExists checks if a location exists.
func (b *Backend) LocationExists(ctx context.Context, name string, _ vaultmux.Session) (bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	_, ok := b.locations[name]
	return ok, nil
}

// CreateLocation creates a new location.
func (b *Backend) CreateLocation(ctx context.Context, name string, _ vaultmux.Session) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.locations[name] {
		return vaultmux.ErrAlreadyExists
	}

	b.locations[name] = true
	return nil
}

// ListItemsInLocation lists items in a specific location.
func (b *Backend) ListItemsInLocation(ctx context.Context, locType, locValue string, _ vaultmux.Session) ([]*vaultmux.Item, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var items []*vaultmux.Item
	for _, item := range b.items {
		if item.Location == locValue {
			itemCopy := *item
			items = append(items, &itemCopy)
		}
	}

	return items, nil
}

// Helper methods for tests

// SetItem directly sets an item in the store (for test setup).
func (b *Backend) SetItem(name, content string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	b.items[name] = &vaultmux.Item{
		ID:       name,
		Name:     name,
		Type:     vaultmux.ItemTypeSecureNote,
		Notes:    content,
		Created:  now,
		Modified: now,
	}
}

// SetItemWithLocation sets an item with a specific location.
func (b *Backend) SetItemWithLocation(name, content, location string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	b.items[name] = &vaultmux.Item{
		ID:       name,
		Name:     name,
		Type:     vaultmux.ItemTypeSecureNote,
		Notes:    content,
		Location: location,
		Created:  now,
		Modified: now,
	}
}

// Clear removes all items and locations.
func (b *Backend) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.items = make(map[string]*vaultmux.Item)
	b.locations = make(map[string]bool)
}

// mockSession implements vaultmux.Session for testing.
type mockSession struct{}

func (s *mockSession) Token() string                     { return "mock-token" }
func (s *mockSession) IsValid(ctx context.Context) bool  { return true }
func (s *mockSession) Refresh(ctx context.Context) error { return nil }
func (s *mockSession) ExpiresAt() time.Time              { return time.Time{} }
