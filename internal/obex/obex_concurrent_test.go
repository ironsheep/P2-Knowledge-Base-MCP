package obex

import (
	"sync"
	"testing"
)

// TestConcurrentReadsAndClears runs every OBEX entry point at once over one
// manager; under -race it proves no data race and no deadlock between the
// OBEX lock and the index and cache managers it calls.
func TestConcurrentReadsAndClears(t *testing.T) {
	m, r := newTestManager(t)
	addTypicalObjects(r)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(6)
		go func() {
			defer wg.Done()
			if _, err := m.GetObject("2811"); err != nil {
				t.Errorf("GetObject: %v", err)
			}
		}()
		go func() {
			defer wg.Done()
			if ids := m.GetObjectIDs(); len(ids) != 2 {
				t.Errorf("GetObjectIDs = %v", ids)
			}
		}()
		go func() { defer wg.Done(); _, _ = m.Search("motor", "", "", 10) }()
		go func() { defer wg.Done(); _ = m.GetTotalObjects() }()
		go func() { defer wg.Done(); _ = m.ClearCache() }()
		go func() { defer wg.Done(); _, _ = m.GetCacheStats() }()
	}
	wg.Wait()
}
