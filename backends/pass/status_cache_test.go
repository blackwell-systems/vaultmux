package pass

import (
	"sync"
	"testing"
	"time"
)

func TestStatusCache_GetSet(t *testing.T) {
	var sc statusCache

	// Initially expired (zero timestamp)
	_, valid := sc.get(5 * time.Second)
	if valid {
		t.Error("get() on empty cache should return invalid")
	}

	// Set authenticated=true
	sc.set(true)

	// Should be valid immediately
	result, valid := sc.get(5 * time.Second)
	if !valid {
		t.Error("get() should return valid immediately after set()")
	}
	if !result {
		t.Error("get() should return true after set(true)")
	}

	// Still valid within TTL
	time.Sleep(2 * time.Second)
	result, valid = sc.get(5 * time.Second)
	if !valid {
		t.Error("get() should be valid within TTL")
	}
	if !result {
		t.Error("get() should return true")
	}
}

func TestStatusCache_Expiration(t *testing.T) {
	var sc statusCache

	sc.set(true)

	// Valid within 1 second TTL
	result, valid := sc.get(1 * time.Second)
	if !valid || !result {
		t.Error("get() should be valid within TTL")
	}

	// Wait for expiration
	time.Sleep(1100 * time.Millisecond)

	// Should be expired
	_, valid = sc.get(1 * time.Second)
	if valid {
		t.Error("get() should be invalid after TTL expires")
	}
}

func TestStatusCache_Concurrent(t *testing.T) {
	var sc statusCache
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(val bool) {
			defer wg.Done()
			sc.set(val)
		}(i%2 == 0)
	}

	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = sc.get(5 * time.Second)
		}()
	}

	wg.Wait()

	// Should not panic or race (verified with -race flag)
}

func TestStatusCache_AlternatingStates(t *testing.T) {
	var sc statusCache

	// Set true
	sc.set(true)
	result, valid := sc.get(5 * time.Second)
	if !valid || !result {
		t.Error("Expected true")
	}

	// Set false
	sc.set(false)
	result, valid = sc.get(5 * time.Second)
	if !valid || result {
		t.Error("Expected false")
	}

	// Set true again
	sc.set(true)
	result, valid = sc.get(5 * time.Second)
	if !valid || !result {
		t.Error("Expected true")
	}
}
