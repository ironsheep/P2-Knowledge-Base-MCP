package cache

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ironsheep/p2kb-mcp/internal/filter"
	"github.com/ironsheep/p2kb-mcp/internal/kbtest"
	"github.com/ironsheep/p2kb-mcp/internal/logging"
)

// rawBody carries a field each test rule removes differently: ruleA removes
// alpha, ruleB removes beta, BuiltinRule removes neither.
const rawBody = "keep: 1\nalpha: a\nbeta: b\n"

var (
	ruleA = filter.Rule{Format: 1, RemoveFields: []string{"alpha"}}
	ruleB = filter.Rule{Format: 1, RemoveFields: []string{"beta"}}
)

// newStampedManager returns a Manager the way NewManager builds one, over
// cacheDir and a Remote serving rawBody at testContentPath.
func newStampedManager(t *testing.T, cacheDir string, r *kbtest.Remote) *Manager {
	t.Helper()
	return &Manager{fetcher: r.Client(), cacheDir: cacheDir, memory: make(map[string]cacheEntry), rule: filter.BuiltinRule}
}

func remoteWithBody(t *testing.T) *kbtest.Remote {
	t.Helper()
	r := kbtest.NewRemote(t)
	r.Put(testContentPath, rawBody)
	return r
}

func diskBody(t *testing.T, m *Manager, key string) (string, bool) {
	t.Helper()
	data, err := os.ReadFile(m.cachePath(key))
	if err != nil {
		return "", false
	}
	return string(data), true
}

func TestStampFormat(t *testing.T) {
	want := filter.BuiltinRule.RuleID() + "/" + filter.EngineVersion
	if got := Stamp(filter.BuiltinRule); got != want {
		t.Errorf("Stamp = %q, want %q", got, want)
	}
}

func TestFirstSetRuleWritesStamp(t *testing.T) {
	m := newStampedManager(t, t.TempDir(), remoteWithBody(t))
	m.SetRule(ruleA)
	if got := m.readStamp(); got != Stamp(ruleA) {
		t.Errorf("stamp on disk = %q, want %q", got, Stamp(ruleA))
	}
}

func TestBodiesAreFilteredUnderRuleInEffect(t *testing.T) {
	m := newStampedManager(t, t.TempDir(), remoteWithBody(t))
	m.SetRule(ruleA)
	got, err := m.GetOrFetch("k", testContentPath, "", knownMtime)
	if err != nil {
		t.Fatalf("GetOrFetch: %v", err)
	}
	if want := "keep: 1\nbeta: b\n"; got != want {
		t.Errorf("content = %q, want %q (filtered under ruleA)", got, want)
	}
}

func TestSameRuleOnRestartKeepsCacheAndFetchesNothing(t *testing.T) {
	dir := t.TempDir()
	r := remoteWithBody(t)
	m1 := newStampedManager(t, dir, r)
	m1.SetRule(ruleA)
	if _, err := m1.GetOrFetch("k", testContentPath, "", knownMtime); err != nil {
		t.Fatalf("GetOrFetch: %v", err)
	}

	m2 := newStampedManager(t, dir, r) // restart
	m2.SetRule(ruleA)
	got, err := m2.GetOrFetch("k", testContentPath, "", knownMtime)
	if err != nil {
		t.Fatalf("GetOrFetch after restart: %v", err)
	}
	if hits := r.Hits(testContentPath); hits != 1 {
		t.Errorf("remote fetches = %d, want 1 (same rule must keep the cache)", hits)
	}
	if want := filter.Apply(ruleA, rawBody); got != want {
		t.Errorf("content = %q, want %q", got, want)
	}
}

func TestDiscardCases(t *testing.T) {
	for _, tc := range []struct {
		name      string
		diskStamp string // "" = no stamp file
		rule      filter.Rule
	}{
		{"different rule", Stamp(ruleA), ruleB},
		{"engine version change alone", ruleA.RuleID() + "/0", ruleA},
		{"no stamp (cache from before stamping)", "", ruleA},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newStampedManager(t, t.TempDir(), remoteWithBody(t))
			if err := m.saveToDisk("old", "sources:\n  - provenance\n", knownMtime); err != nil {
				t.Fatal(err)
			}
			if tc.diskStamp != "" {
				if err := m.writeStamp(tc.diskStamp); err != nil {
					t.Fatal(err)
				}
			}
			m.memory["old"] = cacheEntry{content: "sources: x\n", mtime: knownMtime}

			m.SetRule(tc.rule)

			if _, ok := diskBody(t, m, "old"); ok {
				t.Error("disk body survived the discard")
			}
			if len(m.memory) != 0 {
				t.Errorf("memory entries = %d, want 0", len(m.memory))
			}
			if got := m.readStamp(); got != Stamp(tc.rule) {
				t.Errorf("stamp = %q, want %q", got, Stamp(tc.rule))
			}
		})
	}
}

func TestMatchingStampKeepsBodies(t *testing.T) {
	m := newStampedManager(t, t.TempDir(), remoteWithBody(t))
	if err := m.writeStamp(Stamp(ruleA)); err != nil {
		t.Fatal(err)
	}
	if err := m.saveToDisk("kept", "keep: 1\n", knownMtime); err != nil {
		t.Fatal(err)
	}
	m.SetRule(ruleA)
	if _, ok := diskBody(t, m, "kept"); !ok {
		t.Error("a matching stamp discarded the cache")
	}
}

func TestFlushKeepsStamp(t *testing.T) {
	m := newStampedManager(t, t.TempDir(), remoteWithBody(t))
	m.SetRule(ruleA)
	if _, err := m.GetOrFetch("k", testContentPath, "", knownMtime); err != nil {
		t.Fatal(err)
	}
	m.Clear()
	if _, ok := diskBody(t, m, "k"); ok {
		t.Error("Clear left a body on disk")
	}
	if got := m.readStamp(); got != Stamp(ruleA) {
		t.Errorf("stamp after Clear = %q, want %q", got, Stamp(ruleA))
	}
	if keys := m.GetCachedKeys(); len(keys) != 0 {
		t.Errorf("GetCachedKeys after Clear = %v, want none (the stamp is not a key)", keys)
	}
	if s := m.GetStats(); s.DiskEntries != 0 {
		t.Errorf("DiskEntries after Clear = %d, want 0", s.DiskEntries)
	}
}

func TestUnwritableStampIsLoggedAndTreatedAsUnstamped(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	var buf bytes.Buffer
	prevOut := log.Writer()
	log.SetOutput(&buf)
	prevLevel := logging.SetLevel(logging.Warn)
	t.Cleanup(func() { log.SetOutput(prevOut); logging.SetLevel(prevLevel) })

	dir := t.TempDir()
	if err := os.Chmod(dir, 0555); err != nil { // cache/ cannot be created
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0755) })

	r := remoteWithBody(t)
	m := newStampedManager(t, dir, r)
	m.SetRule(ruleA)
	if !strings.Contains(buf.String(), "failed to write cache stamp") {
		t.Errorf("log = %q, want the stamp-write failure", buf.String())
	}

	// Next start: the directory is writable again and holds a body, but no
	// stamp was ever written, so the cache is treated as unstamped.
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	m2 := newStampedManager(t, dir, r)
	if err := m2.saveToDisk("k", "unstamped body\n", knownMtime); err != nil {
		t.Fatal(err)
	}
	m2.SetRule(ruleA)
	if _, ok := diskBody(t, m2, "k"); ok {
		t.Error("a cache with no stamp was kept")
	}
}

// TestRuleChangeDuringFetchStoresNewFiltering is the race guard: a fetch held
// mid-flight while SetRule discards must end with the caller, memory and
// disk all holding the body filtered under the NEW rule.
func TestRuleChangeDuringFetchStoresNewFiltering(t *testing.T) {
	r := remoteWithBody(t)
	m := newStampedManager(t, t.TempDir(), r)
	m.SetRule(ruleA)
	gate := r.Gate(testContentPath)

	type result struct {
		content string
		err     error
	}
	done := make(chan result, 1)
	go func() {
		c, err := m.GetOrFetch("k", testContentPath, "", knownMtime)
		done <- result{c, err}
	}()

	select {
	case <-gate.Arrived():
	case <-time.After(5 * time.Second):
		t.Fatal("fetch never reached the gate")
	}
	m.SetRule(ruleB) // discard while the fetch is in flight
	gate.Release()

	var res result
	select {
	case res = <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("fetch did not finish after release")
	}
	if res.err != nil {
		t.Fatalf("GetOrFetch: %v", res.err)
	}

	want := filter.Apply(ruleB, rawBody)
	if res.content != want {
		t.Errorf("returned %q, want %q (new rule)", res.content, want)
	}
	m.mu.RLock()
	mem := m.memory["k"].content
	m.mu.RUnlock()
	if mem != want {
		t.Errorf("memory holds %q, want %q", mem, want)
	}
	if disk, _ := diskBody(t, m, "k"); disk != want {
		t.Errorf("disk holds %q, want %q", disk, want)
	}
}

// TestDiskReadOverlappingDiscardIsNotHydrated: a disk-tier read whose
// generation was taken before a discard neither serves nor hydrates the body.
func TestDiskReadOverlappingDiscardIsNotHydrated(t *testing.T) {
	m := newStampedManager(t, t.TempDir(), remoteWithBody(t))
	m.SetRule(ruleA)
	_, gen := m.snapshot() // the read starts here...

	m.SetRule(ruleB) // ...a discard lands...
	if err := m.saveToDisk("k", "body filtered under ruleA\n", knownMtime); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(m.cachePath("k"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := m.readAndHydrate("k", info, gen); err != errSuperseded {
		t.Errorf("readAndHydrate err = %v, want errSuperseded", err)
	}
	m.mu.RLock()
	_, hydrated := m.memory["k"]
	m.mu.RUnlock()
	if hydrated {
		t.Error("a read that overlapped a discard hydrated memory")
	}
}

func TestStampFileLivesInCacheDir(t *testing.T) {
	m := newStampedManager(t, t.TempDir(), remoteWithBody(t))
	if want := filepath.Join(m.cacheDir, "cache", "filter-stamp"); m.stampPath() != want {
		t.Errorf("stampPath = %q, want %q", m.stampPath(), want)
	}
}
