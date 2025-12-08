package azurekeyvault

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/blackwell-systems/vaultmux"
)

// TestIntegration runs full CRUD tests against a real Azure Key Vault.
// Skips if AZURE_VAULT_URL environment variable is not set.
//
// To run these tests:
// 1. Create an Azure Key Vault
// 2. Grant "Key Vault Secrets Officer" role to your identity
// 3. Set AZURE_VAULT_URL environment variable
// 4. Authenticate: az login OR set AZURE_TENANT_ID/AZURE_CLIENT_ID/AZURE_CLIENT_SECRET
// 5. Run: go test -v ./backends/azurekeyvault/
func TestIntegration(t *testing.T) {
	vaultURL := os.Getenv("AZURE_VAULT_URL")
	if vaultURL == "" {
		t.Skip("AZURE_VAULT_URL not set - skipping integration tests")
	}

	ctx := context.Background()

	// Create backend
	backend, err := New(map[string]string{
		"vault_url": vaultURL,
		"prefix":    "vaultmux-test-",
	}, "")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer backend.Close()

	// Initialize
	err = backend.Init(ctx)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Authenticate
	session, err := backend.Authenticate(ctx)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	if !session.IsValid(ctx) {
		t.Fatal("Session is not valid after authentication")
	}

	// Test item name (unique per test run)
	testItem := fmt.Sprintf("integration-test-%d", os.Getpid())

	// Clean up any existing test item
	backend.DeleteItem(ctx, testItem, session)

	// Test CreateItem
	t.Run("CreateItem", func(t *testing.T) {
		err = backend.CreateItem(ctx, testItem, "test-secret-value", session)
		if err != nil {
			t.Errorf("CreateItem() error = %v", err)
		}
	})

	// Test ItemExists (should exist)
	t.Run("ItemExists_True", func(t *testing.T) {
		exists, err := backend.ItemExists(ctx, testItem, session)
		if err != nil {
			t.Errorf("ItemExists() error = %v", err)
		}
		if !exists {
			t.Error("ItemExists() = false, want true")
		}
	})

	// Test GetItem
	t.Run("GetItem", func(t *testing.T) {
		item, err := backend.GetItem(ctx, testItem, session)
		if err != nil {
			t.Errorf("GetItem() error = %v", err)
		}
		if item == nil {
			t.Fatal("GetItem() returned nil item")
		}
		if item.Name != testItem {
			t.Errorf("GetItem() name = %q, want %q", item.Name, testItem)
		}
		if item.Notes != "test-secret-value" {
			t.Errorf("GetItem() notes = %q, want %q", item.Notes, "test-secret-value")
		}
	})

	// Test GetNotes
	t.Run("GetNotes", func(t *testing.T) {
		notes, err := backend.GetNotes(ctx, testItem, session)
		if err != nil {
			t.Errorf("GetNotes() error = %v", err)
		}
		if notes != "test-secret-value" {
			t.Errorf("GetNotes() = %q, want %q", notes, "test-secret-value")
		}
	})

	// Test UpdateItem
	t.Run("UpdateItem", func(t *testing.T) {
		err = backend.UpdateItem(ctx, testItem, "updated-secret-value", session)
		if err != nil {
			t.Errorf("UpdateItem() error = %v", err)
		}

		// Verify update
		notes, err := backend.GetNotes(ctx, testItem, session)
		if err != nil {
			t.Errorf("GetNotes() after update error = %v", err)
		}
		if notes != "updated-secret-value" {
			t.Errorf("GetNotes() after update = %q, want %q", notes, "updated-secret-value")
		}
	})

	// Test ListItems
	t.Run("ListItems", func(t *testing.T) {
		items, err := backend.ListItems(ctx, session)
		if err != nil {
			t.Errorf("ListItems() error = %v", err)
		}

		found := false
		for _, item := range items {
			if item.Name == testItem {
				found = true
				break
			}
		}
		if !found {
			t.Error("ListItems() did not include test item")
		}
	})

	// Test DeleteItem
	t.Run("DeleteItem", func(t *testing.T) {
		err = backend.DeleteItem(ctx, testItem, session)
		if err != nil {
			t.Errorf("DeleteItem() error = %v", err)
		}

		// Verify deletion
		exists, err := backend.ItemExists(ctx, testItem, session)
		if err != nil {
			t.Errorf("ItemExists() after delete error = %v", err)
		}
		if exists {
			t.Error("ItemExists() after delete = true, want false")
		}
	})

	// Test error cases
	t.Run("GetItem_NotFound", func(t *testing.T) {
		_, err := backend.GetItem(ctx, "nonexistent-item", session)
		if err == nil {
			t.Error("GetItem() for nonexistent item expected error, got nil")
		}
		if err != nil && err != vaultmux.ErrNotFound {
			t.Errorf("GetItem() error = %v, want ErrNotFound", err)
		}
	})

	t.Run("UpdateItem_NotFound", func(t *testing.T) {
		err := backend.UpdateItem(ctx, "nonexistent-item", "value", session)
		if err == nil {
			t.Error("UpdateItem() for nonexistent item expected error, got nil")
		}
		if err != nil && err != vaultmux.ErrNotFound {
			t.Errorf("UpdateItem() error = %v, want ErrNotFound", err)
		}
	})

	t.Run("DeleteItem_NotFound", func(t *testing.T) {
		err := backend.DeleteItem(ctx, "nonexistent-item", session)
		if err == nil {
			t.Error("DeleteItem() for nonexistent item expected error, got nil")
		}
		if err != nil && err != vaultmux.ErrNotFound {
			t.Errorf("DeleteItem() error = %v, want ErrNotFound", err)
		}
	})

	// Test CreateItem with duplicate
	t.Run("CreateItem_AlreadyExists", func(t *testing.T) {
		// Create first time
		itemName := fmt.Sprintf("duplicate-test-%d", os.Getpid())
		err = backend.CreateItem(ctx, itemName, "value", session)
		if err != nil {
			t.Fatalf("CreateItem() first call error = %v", err)
		}
		defer backend.DeleteItem(ctx, itemName, session)

		// Try to create again
		err = backend.CreateItem(ctx, itemName, "value2", session)
		if err == nil {
			t.Error("CreateItem() for existing item expected error, got nil")
		}
		if err != nil && err != vaultmux.ErrAlreadyExists {
			t.Errorf("CreateItem() error = %v, want ErrAlreadyExists", err)
		}
	})
}

// TestIntegration_Pagination tests listing with many secrets.
// Only runs if AZURE_VAULT_URL is set and vault has multiple secrets.
func TestIntegration_Pagination(t *testing.T) {
	vaultURL := os.Getenv("AZURE_VAULT_URL")
	if vaultURL == "" {
		t.Skip("AZURE_VAULT_URL not set - skipping pagination test")
	}

	ctx := context.Background()

	backend, err := New(map[string]string{
		"vault_url": vaultURL,
		"prefix":    "vaultmux-test-",
	}, "")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer backend.Close()

	err = backend.Init(ctx)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	session, err := backend.Authenticate(ctx)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	// Create multiple test items
	numItems := 5
	createdItems := make([]string, numItems)

	for i := 0; i < numItems; i++ {
		itemName := fmt.Sprintf("pagination-test-%d-%d", os.Getpid(), i)
		createdItems[i] = itemName

		err = backend.CreateItem(ctx, itemName, fmt.Sprintf("value-%d", i), session)
		if err != nil {
			t.Logf("CreateItem(%d) error = %v (may already exist)", i, err)
		}
	}

	// Clean up after test
	defer func() {
		for _, itemName := range createdItems {
			backend.DeleteItem(ctx, itemName, session)
		}
	}()

	// List all items
	items, err := backend.ListItems(ctx, session)
	if err != nil {
		t.Fatalf("ListItems() error = %v", err)
	}

	// Verify we got at least our created items
	foundCount := 0
	for _, createdItem := range createdItems {
		for _, item := range items {
			if item.Name == createdItem {
				foundCount++
				break
			}
		}
	}

	if foundCount < numItems {
		t.Errorf("ListItems() found %d/%d created items", foundCount, numItems)
	}

	t.Logf("Successfully listed %d total items, including %d created items", len(items), foundCount)
}
