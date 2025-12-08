package gcpsecrets

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/blackwell-systems/vaultmux"
)

// TestIntegration runs complete CRUD tests against a real GCP project.
//
// Prerequisites:
//   - GCP project with Secret Manager API enabled
//   - Service account with secretmanager.admin role
//   - GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
//
// Run with:
//
//	GOOGLE_APPLICATION_CREDENTIALS=/path/to/sa.json \
//	GCP_PROJECT_ID=my-project \
//	go test -v ./backends/gcpsecrets/
//
// Note: Unlike AWS (LocalStack) or Windows (WSL2), GCP has no good local emulator.
// These tests require a real GCP project or will skip gracefully.
func TestIntegration(t *testing.T) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		t.Skip("GCP_PROJECT_ID not set - skipping integration tests")
	}

	backend, err := New(map[string]string{
		"project_id": projectID,
		"prefix":     "test-vaultmux-",
	}, "")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// Init
	t.Run("Init", func(t *testing.T) {
		if err := backend.Init(ctx); err != nil {
			t.Fatalf("Init() error = %v", err)
		}
	})

	// Authenticate
	var session vaultmux.Session
	t.Run("Authenticate", func(t *testing.T) {
		var err error
		session, err = backend.Authenticate(ctx)
		if err != nil {
			t.Fatalf("Authenticate() error = %v (check GOOGLE_APPLICATION_CREDENTIALS)", err)
		}

		if !session.IsValid(ctx) {
			t.Error("Session is not valid after authentication")
		}
	})

	// IsAuthenticated
	t.Run("IsAuthenticated", func(t *testing.T) {
		if !backend.IsAuthenticated(ctx) {
			t.Error("IsAuthenticated() = false, want true")
		}
	})

	itemName := "test-item"
	itemContent := "test-secret-content"

	// CreateItem
	t.Run("CreateItem", func(t *testing.T) {
		err := backend.CreateItem(ctx, itemName, itemContent, session)
		if err != nil {
			t.Fatalf("CreateItem() error = %v", err)
		}
	})

	// CreateItem duplicate (should fail)
	t.Run("CreateItem_AlreadyExists", func(t *testing.T) {
		err := backend.CreateItem(ctx, itemName, itemContent, session)
		if !errors.Is(err, vaultmux.ErrAlreadyExists) {
			t.Errorf("CreateItem() duplicate error = %v, want ErrAlreadyExists", err)
		}
	})

	// ItemExists
	t.Run("ItemExists", func(t *testing.T) {
		exists, err := backend.ItemExists(ctx, itemName, session)
		if err != nil {
			t.Fatalf("ItemExists() error = %v", err)
		}
		if !exists {
			t.Error("ItemExists() = false, want true")
		}
	})

	// GetItem
	t.Run("GetItem", func(t *testing.T) {
		item, err := backend.GetItem(ctx, itemName, session)
		if err != nil {
			t.Fatalf("GetItem() error = %v", err)
		}

		if item.Name != itemName {
			t.Errorf("GetItem().Name = %q, want %q", item.Name, itemName)
		}

		if item.Notes != itemContent {
			t.Errorf("GetItem().Notes = %q, want %q", item.Notes, itemContent)
		}

		if item.Type != vaultmux.ItemTypeSecureNote {
			t.Errorf("GetItem().Type = %v, want ItemTypeSecureNote", item.Type)
		}

		if item.ID == "" {
			t.Error("GetItem().ID is empty, want non-empty resource name")
		}
	})

	// GetNotes
	t.Run("GetNotes", func(t *testing.T) {
		notes, err := backend.GetNotes(ctx, itemName, session)
		if err != nil {
			t.Fatalf("GetNotes() error = %v", err)
		}

		if notes != itemContent {
			t.Errorf("GetNotes() = %q, want %q", notes, itemContent)
		}
	})

	// ListItems
	t.Run("ListItems", func(t *testing.T) {
		items, err := backend.ListItems(ctx, session)
		if err != nil {
			t.Fatalf("ListItems() error = %v", err)
		}

		// Should have at least our test item
		found := false
		for _, item := range items {
			if item.Name == itemName {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("ListItems() did not include %q", itemName)
		}
	})

	// UpdateItem
	updatedContent := "updated-secret-content"
	t.Run("UpdateItem", func(t *testing.T) {
		err := backend.UpdateItem(ctx, itemName, updatedContent, session)
		if err != nil {
			t.Fatalf("UpdateItem() error = %v", err)
		}

		// Verify update
		notes, err := backend.GetNotes(ctx, itemName, session)
		if err != nil {
			t.Fatalf("GetNotes() after update error = %v", err)
		}

		if notes != updatedContent {
			t.Errorf("GetNotes() after update = %q, want %q", notes, updatedContent)
		}
	})

	// UpdateItem non-existent (should fail)
	t.Run("UpdateItem_NotFound", func(t *testing.T) {
		err := backend.UpdateItem(ctx, "nonexistent", "content", session)
		if !errors.Is(err, vaultmux.ErrNotFound) {
			t.Errorf("UpdateItem() non-existent error = %v, want ErrNotFound", err)
		}
	})

	// DeleteItem
	t.Run("DeleteItem", func(t *testing.T) {
		err := backend.DeleteItem(ctx, itemName, session)
		if err != nil {
			t.Fatalf("DeleteItem() error = %v", err)
		}

		// Verify deletion
		exists, err := backend.ItemExists(ctx, itemName, session)
		if err != nil {
			t.Fatalf("ItemExists() after delete error = %v", err)
		}

		if exists {
			t.Error("ItemExists() after delete = true, want false")
		}
	})

	// DeleteItem non-existent (should fail)
	t.Run("DeleteItem_NotFound", func(t *testing.T) {
		err := backend.DeleteItem(ctx, "nonexistent", session)
		if !errors.Is(err, vaultmux.ErrNotFound) {
			t.Errorf("DeleteItem() non-existent error = %v, want ErrNotFound", err)
		}
	})

	// GetItem non-existent (should fail)
	t.Run("GetItem_NotFound", func(t *testing.T) {
		_, err := backend.GetItem(ctx, "nonexistent", session)
		if !errors.Is(err, vaultmux.ErrNotFound) {
			t.Errorf("GetItem() non-existent error = %v, want ErrNotFound", err)
		}
	})

	// GetNotes non-existent (should fail)
	t.Run("GetNotes_NotFound", func(t *testing.T) {
		_, err := backend.GetNotes(ctx, "nonexistent", session)
		if !errors.Is(err, vaultmux.ErrNotFound) {
			t.Errorf("GetNotes() non-existent error = %v, want ErrNotFound", err)
		}
	})

	// ItemExists for non-existent item
	t.Run("ItemExists_False", func(t *testing.T) {
		exists, err := backend.ItemExists(ctx, "nonexistent", session)
		if err != nil {
			t.Fatalf("ItemExists() error = %v", err)
		}
		if exists {
			t.Error("ItemExists() for nonexistent = true, want false")
		}
	})
}

// TestIntegration_Pagination tests that large secret collections are handled correctly.
func TestIntegration_Pagination(t *testing.T) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		t.Skip("GCP_PROJECT_ID not set - skipping pagination test")
	}

	backend, err := New(map[string]string{
		"project_id": projectID,
		"prefix":     "pagination-test-",
	}, "")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	err = backend.Init(ctx)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	session, err := backend.Authenticate(ctx)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	// Create 5 test items (GCP is fast, but real API so keep count low)
	itemCount := 5
	for i := 0; i < itemCount; i++ {
		itemName := fmt.Sprintf("item-%d", i)
		err = backend.CreateItem(ctx, itemName, "content", session)
		if err != nil {
			t.Fatalf("CreateItem(%q) error = %v", itemName, err)
		}
	}

	// List all items
	items, err := backend.ListItems(ctx, session)
	if err != nil {
		t.Fatalf("ListItems() error = %v", err)
	}

	if len(items) < itemCount {
		t.Errorf("ListItems() returned %d items, want at least %d", len(items), itemCount)
	}

	// Cleanup
	for i := 0; i < itemCount; i++ {
		itemName := fmt.Sprintf("item-%d", i)
		_ = backend.DeleteItem(ctx, itemName, session)
	}
}
