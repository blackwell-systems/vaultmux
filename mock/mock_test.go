package mock

import (
	"context"
	"errors"
	"testing"

	"github.com/blackwell-systems/vaultmux"
)

func TestMockBackend_Name(t *testing.T) {
	backend := New()
	if got := backend.Name(); got != "mock" {
		t.Errorf("Name() = %q, want %q", got, "mock")
	}
}

func TestMockBackend_Init(t *testing.T) {
	backend := New()
	ctx := context.Background()

	if err := backend.Init(ctx); err != nil {
		t.Errorf("Init() error = %v, want nil", err)
	}
}

func TestMockBackend_Authenticate(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		backend := New()
		session, err := backend.Authenticate(ctx)
		if err != nil {
			t.Fatalf("Authenticate() error = %v, want nil", err)
		}
		if session == nil {
			t.Fatal("Authenticate() returned nil session")
		}
		if session.Token() != "mock-token" {
			t.Errorf("session.Token() = %q, want %q", session.Token(), "mock-token")
		}
	})

	t.Run("auth error", func(t *testing.T) {
		backend := New()
		backend.AuthError = errors.New("auth failed")

		session, err := backend.Authenticate(ctx)
		if err == nil {
			t.Fatal("Authenticate() error = nil, want error")
		}
		if session != nil {
			t.Errorf("Authenticate() session = %v, want nil", session)
		}
	})
}

func TestMockBackend_CRUD(t *testing.T) {
	ctx := context.Background()
	backend := New()
	session, _ := backend.Authenticate(ctx)

	t.Run("create item", func(t *testing.T) {
		err := backend.CreateItem(ctx, "test-key", "test-value", session)
		if err != nil {
			t.Fatalf("CreateItem() error = %v, want nil", err)
		}

		// Verify it exists
		exists, err := backend.ItemExists(ctx, "test-key", session)
		if err != nil {
			t.Fatalf("ItemExists() error = %v", err)
		}
		if !exists {
			t.Error("ItemExists() = false, want true")
		}
	})

	t.Run("get item", func(t *testing.T) {
		backend.SetItem("get-test", "get-value")

		item, err := backend.GetItem(ctx, "get-test", session)
		if err != nil {
			t.Fatalf("GetItem() error = %v, want nil", err)
		}
		if item.Name != "get-test" {
			t.Errorf("item.Name = %q, want %q", item.Name, "get-test")
		}
		if item.Notes != "get-value" {
			t.Errorf("item.Notes = %q, want %q", item.Notes, "get-value")
		}
	})

	t.Run("get notes", func(t *testing.T) {
		backend.SetItem("notes-test", "notes-value")

		notes, err := backend.GetNotes(ctx, "notes-test", session)
		if err != nil {
			t.Fatalf("GetNotes() error = %v, want nil", err)
		}
		if notes != "notes-value" {
			t.Errorf("GetNotes() = %q, want %q", notes, "notes-value")
		}
	})

	t.Run("update item", func(t *testing.T) {
		backend.SetItem("update-test", "old-value")

		err := backend.UpdateItem(ctx, "update-test", "new-value", session)
		if err != nil {
			t.Fatalf("UpdateItem() error = %v, want nil", err)
		}

		notes, _ := backend.GetNotes(ctx, "update-test", session)
		if notes != "new-value" {
			t.Errorf("after update, notes = %q, want %q", notes, "new-value")
		}
	})

	t.Run("delete item", func(t *testing.T) {
		backend.SetItem("delete-test", "delete-value")

		err := backend.DeleteItem(ctx, "delete-test", session)
		if err != nil {
			t.Fatalf("DeleteItem() error = %v, want nil", err)
		}

		exists, _ := backend.ItemExists(ctx, "delete-test", session)
		if exists {
			t.Error("ItemExists() = true after delete, want false")
		}
	})

	t.Run("list items", func(t *testing.T) {
		backend.Clear()
		backend.SetItem("item1", "value1")
		backend.SetItem("item2", "value2")
		backend.SetItem("item3", "value3")

		items, err := backend.ListItems(ctx, session)
		if err != nil {
			t.Fatalf("ListItems() error = %v, want nil", err)
		}
		if len(items) != 3 {
			t.Errorf("ListItems() returned %d items, want 3", len(items))
		}
	})
}

func TestMockBackend_Errors(t *testing.T) {
	ctx := context.Background()
	backend := New()
	session, _ := backend.Authenticate(ctx)

	t.Run("get error", func(t *testing.T) {
		backend.GetError = errors.New("get failed")
		_, err := backend.GetItem(ctx, "test", session)
		if err == nil {
			t.Error("GetItem() error = nil, want error")
		}
	})

	t.Run("create error", func(t *testing.T) {
		backend.CreateError = errors.New("create failed")
		err := backend.CreateItem(ctx, "test", "value", session)
		if err == nil {
			t.Error("CreateItem() error = nil, want error")
		}
	})

	t.Run("update error", func(t *testing.T) {
		backend.SetItem("test", "value")
		backend.UpdateError = errors.New("update failed")
		err := backend.UpdateItem(ctx, "test", "new", session)
		if err == nil {
			t.Error("UpdateItem() error = nil, want error")
		}
	})

	t.Run("delete error", func(t *testing.T) {
		backend.SetItem("test", "value")
		backend.DeleteError = errors.New("delete failed")
		err := backend.DeleteItem(ctx, "test", session)
		if err == nil {
			t.Error("DeleteItem() error = nil, want error")
		}
	})

	t.Run("not found", func(t *testing.T) {
		backend.GetError = nil
		_, err := backend.GetItem(ctx, "nonexistent", session)
		if !errors.Is(err, vaultmux.ErrNotFound) {
			t.Errorf("GetItem() error = %v, want ErrNotFound", err)
		}
	})

	t.Run("already exists", func(t *testing.T) {
		backend.CreateError = nil
		backend.SetItem("existing", "value")
		err := backend.CreateItem(ctx, "existing", "new", session)
		if !errors.Is(err, vaultmux.ErrAlreadyExists) {
			t.Errorf("CreateItem() error = %v, want ErrAlreadyExists", err)
		}
	})
}

func TestMockBackend_Locations(t *testing.T) {
	ctx := context.Background()
	backend := New()
	session, _ := backend.Authenticate(ctx)

	t.Run("create location", func(t *testing.T) {
		err := backend.CreateLocation(ctx, "test-folder", session)
		if err != nil {
			t.Fatalf("CreateLocation() error = %v, want nil", err)
		}

		exists, err := backend.LocationExists(ctx, "test-folder", session)
		if err != nil {
			t.Fatalf("LocationExists() error = %v", err)
		}
		if !exists {
			t.Error("LocationExists() = false, want true")
		}
	})

	t.Run("list locations", func(t *testing.T) {
		_ = backend.CreateLocation(ctx, "folder1", session)
		_ = backend.CreateLocation(ctx, "folder2", session)

		locations, err := backend.ListLocations(ctx, session)
		if err != nil {
			t.Fatalf("ListLocations() error = %v", err)
		}
		if len(locations) < 2 {
			t.Errorf("ListLocations() returned %d, want at least 2", len(locations))
		}
	})

	t.Run("list items in location", func(t *testing.T) {
		backend.SetItemWithLocation("item1", "value1", "work")
		backend.SetItemWithLocation("item2", "value2", "work")
		backend.SetItemWithLocation("item3", "value3", "personal")

		items, err := backend.ListItemsInLocation(ctx, "folder", "work", session)
		if err != nil {
			t.Fatalf("ListItemsInLocation() error = %v", err)
		}
		if len(items) != 2 {
			t.Errorf("ListItemsInLocation() returned %d items, want 2", len(items))
		}
	})
}

func TestMockBackend_Sync(t *testing.T) {
	ctx := context.Background()
	backend := New()
	session, _ := backend.Authenticate(ctx)

	t.Run("success", func(t *testing.T) {
		err := backend.Sync(ctx, session)
		if err != nil {
			t.Errorf("Sync() error = %v, want nil", err)
		}
	})

	t.Run("error", func(t *testing.T) {
		backend.SyncError = errors.New("sync failed")
		err := backend.Sync(ctx, session)
		if err == nil {
			t.Error("Sync() error = nil, want error")
		}
	})
}

func TestMockBackend_Clear(t *testing.T) {
	backend := New()
	backend.SetItem("item1", "value1")
	backend.SetItem("item2", "value2")
	_ = backend.CreateLocation(context.Background(), "folder", nil)

	backend.Clear()

	items, _ := backend.ListItems(context.Background(), nil)
	if len(items) != 0 {
		t.Errorf("after Clear(), ListItems() returned %d items, want 0", len(items))
	}

	locations, _ := backend.ListLocations(context.Background(), nil)
	if len(locations) != 0 {
		t.Errorf("after Clear(), ListLocations() returned %d, want 0", len(locations))
	}
}

func TestMockSession(t *testing.T) {
	ctx := context.Background()
	backend := New()
	session, _ := backend.Authenticate(ctx)

	if token := session.Token(); token != "mock-token" {
		t.Errorf("Token() = %q, want %q", token, "mock-token")
	}

	if !session.IsValid(ctx) {
		t.Error("IsValid() = false, want true")
	}

	if err := session.Refresh(ctx); err != nil {
		t.Errorf("Refresh() error = %v, want nil", err)
	}

	if !session.ExpiresAt().IsZero() {
		t.Error("ExpiresAt() is not zero, want zero time")
	}
}
