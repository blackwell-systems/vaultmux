package vaultmux_test

import (
	"context"
	"fmt"
	"log"

	"github.com/blackwell-systems/vaultmux"
	_ "github.com/blackwell-systems/vaultmux/backends/bitwarden"   // Register backend
	_ "github.com/blackwell-systems/vaultmux/backends/onepassword" // Register backend
	_ "github.com/blackwell-systems/vaultmux/backends/pass"        // Register backend
	"github.com/blackwell-systems/vaultmux/mock"
)

// Example demonstrates basic usage of vaultmux with the mock backend
func Example() {
	ctx := context.Background()

	// Create a mock backend for testing
	backend := mock.New()
	defer func() { _ = backend.Close() }()

	// Initialize
	if err := backend.Init(ctx); err != nil {
		log.Fatal(err)
	}

	// Authenticate
	session, err := backend.Authenticate(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Create an item
	err = backend.CreateItem(ctx, "API-Key", "sk-1234567890", session)
	if err != nil {
		log.Fatal(err)
	}

	// Retrieve the item
	notes, err := backend.GetNotes(ctx, "API-Key", session)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(notes)
	// Output: sk-1234567890
}

// ExampleConfig demonstrates various configuration options
func ExampleConfig() {
	// Basic configuration for pass backend
	cfg := vaultmux.Config{
		Backend: vaultmux.BackendPass,
		Prefix:  "myapp",
	}

	backend, err := vaultmux.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = backend.Close() }()

	fmt.Println(backend.Name())
	// Output: pass
}

// ExampleBackend_ListItems demonstrates listing all items
func ExampleBackend_ListItems() {
	ctx := context.Background()

	// Create mock backend with test data
	backend := mock.New()
	backend.SetItem("key1", "value1")
	backend.SetItem("key2", "value2")

	session, _ := backend.Authenticate(ctx)

	items, err := backend.ListItems(ctx, session)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d items\n", len(items))
	// Output: Found 2 items
}

// ExampleBackend_CreateItem demonstrates creating a new item
func Example_createItem() {
	ctx := context.Background()
	backend := mock.New()
	session, _ := backend.Authenticate(ctx)

	// Create an item
	err := backend.CreateItem(ctx, "password", "secret123", session)
	if err != nil {
		log.Fatal(err)
	}

	// Verify it exists
	exists, _ := backend.ItemExists(ctx, "password", session)
	fmt.Println(exists)
	// Output: true
}

// ExampleBackend_UpdateItem demonstrates updating an existing item
func Example_updateItem() {
	ctx := context.Background()
	backend := mock.New()
	backend.SetItem("token", "old-value")
	session, _ := backend.Authenticate(ctx)

	// Update the item
	err := backend.UpdateItem(ctx, "token", "new-value", session)
	if err != nil {
		log.Fatal(err)
	}

	// Verify the update
	notes, _ := backend.GetNotes(ctx, "token", session)
	fmt.Println(notes)
	// Output: new-value
}

// ExampleBackend_DeleteItem demonstrates deleting an item
func Example_deleteItem() {
	ctx := context.Background()
	backend := mock.New()
	backend.SetItem("temp", "value")
	session, _ := backend.Authenticate(ctx)

	// Delete the item
	err := backend.DeleteItem(ctx, "temp", session)
	if err != nil {
		log.Fatal(err)
	}

	// Verify it's gone
	exists, _ := backend.ItemExists(ctx, "temp", session)
	fmt.Println(exists)
	// Output: false
}

// ExampleBackend_GetNotes demonstrates error handling
func Example_errorHandling() {
	ctx := context.Background()
	backend := mock.New()
	session, _ := backend.Authenticate(ctx)

	// Try to get a non-existent item
	_, err := backend.GetNotes(ctx, "nonexistent", session)

	if err == vaultmux.ErrNotFound {
		fmt.Println("Item not found")
	}
	// Output: Item not found
}

// ExampleBackend_ListLocations demonstrates working with locations
func Example_locations() {
	ctx := context.Background()
	backend := mock.New()
	session, _ := backend.Authenticate(ctx)

	// Create a location
	if err := backend.CreateLocation(ctx, "work", session); err != nil {
		log.Fatal(err)
	}

	// List all locations
	locations, _ := backend.ListLocations(ctx, session)
	fmt.Printf("Locations: %d\n", len(locations))
	// Output: Locations: 1
}
