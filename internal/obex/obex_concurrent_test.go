package obex

import (
	"sync"
	"testing"
	"time"
)

// TestConcurrentEnsureIndex tests the fix for the lock-during-network-I/O issue.
// This tests that multiple goroutines can safely call EnsureIndex concurrently.
func TestConcurrentEnsureIndex(t *testing.T) {
	m := &Manager{
		cacheDir:   t.TempDir(),
		objects:    make(map[string]*OBEXObject),
		objectIDs:  []string{"1234", "5678", "9012"},
		ttl:        1 * time.Hour,
		lastRefresh: time.Now(),
	}

	// Launch 20 concurrent EnsureIndex calls
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// EnsureIndex should return immediately since we have a fresh index
			err := m.EnsureIndex()
			if err != nil {
				t.Errorf("EnsureIndex failed: %v", err)
			}
		}()
	}

	wg.Wait()
	t.Log("Concurrent EnsureIndex calls completed without deadlock")
}

// TestConcurrentGetObjectIDs tests concurrent access to GetObjectIDs.
func TestConcurrentGetObjectIDs(t *testing.T) {
	m := &Manager{
		cacheDir:    t.TempDir(),
		objects:     make(map[string]*OBEXObject),
		objectIDs:   []string{"1234", "5678", "9012", "3456", "7890"},
		ttl:         1 * time.Hour,
		lastRefresh: time.Now(),
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ids := m.GetObjectIDs()
			if len(ids) != 5 {
				t.Errorf("GetObjectIDs returned %d IDs, want 5", len(ids))
			}
		}()
	}

	wg.Wait()
	t.Log("Concurrent GetObjectIDs calls completed without deadlock")
}

// TestConcurrentObjectExists tests concurrent access to objectExists.
func TestConcurrentObjectExists(t *testing.T) {
	m := &Manager{
		cacheDir:    t.TempDir(),
		objects:     make(map[string]*OBEXObject),
		objectIDs:   []string{"1234", "5678", "9012"},
		ttl:         1 * time.Hour,
		lastRefresh: time.Now(),
	}

	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			// Alternate between existing and non-existing IDs
			if idx%2 == 0 {
				if !m.objectExists("1234") {
					t.Error("objectExists(1234) = false, want true")
				}
			} else {
				if m.objectExists("9999") {
					t.Error("objectExists(9999) = true, want false")
				}
			}
		}(i)
	}

	wg.Wait()
	t.Log("Concurrent objectExists calls completed without deadlock")
}

// TestConcurrentGetTotalObjects tests concurrent access to GetTotalObjects.
func TestConcurrentGetTotalObjects(t *testing.T) {
	m := &Manager{
		cacheDir:    t.TempDir(),
		objects:     make(map[string]*OBEXObject),
		objectIDs:   []string{"1234", "5678", "9012"},
		ttl:         1 * time.Hour,
		lastRefresh: time.Now(),
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			total := m.GetTotalObjects()
			if total != 3 {
				t.Errorf("GetTotalObjects = %d, want 3", total)
			}
		}()
	}

	wg.Wait()
	t.Log("Concurrent GetTotalObjects calls completed without deadlock")
}

// TestConcurrentClearCache tests concurrent ClearCache calls don't deadlock.
func TestConcurrentClearCache(t *testing.T) {
	m := &Manager{
		cacheDir:    t.TempDir(),
		objects:     make(map[string]*OBEXObject),
		objectIDs:   []string{"1234", "5678", "9012"},
		ttl:         1 * time.Hour,
		lastRefresh: time.Now(),
	}

	// Pre-populate some objects
	m.objects["1234"] = &OBEXObject{}
	m.objects["5678"] = &OBEXObject{}

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.ClearCache()
		}()
	}

	wg.Wait()
	t.Log("Concurrent ClearCache calls completed without deadlock")
}
