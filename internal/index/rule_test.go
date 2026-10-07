package index

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/ironsheep/p2kb-mcp/internal/fetch"
	"github.com/ironsheep/p2kb-mcp/internal/filter"
	"github.com/ironsheep/p2kb-mcp/internal/kbtest"
)

// indexRuleBlock is a valid format-1 block that differs from BuiltinRule.
const indexRuleBlock = `{"format":1,"remove_fields":["source","x_extra"],"remove_column0_comments":false}`

// indexRule is indexRuleBlock parsed.
var indexRule = filter.Rule{Format: 1, RemoveFields: []string{"source", "x_extra"}, RemoveColumn0Comments: false}

// otherRuleBlock is a second valid block, for rule-change cases.
const otherRuleBlock = `{"format":1,"remove_fields":["verified_against"],"remove_column0_comments":true}`

// format2Block is a block this engine must refuse.
const format2Block = `{"format":2,"remove_fields":["source"],"remove_column0_comments":true}`

// ruleRecorder counts and records the rules delivered to OnRuleChange.
type ruleRecorder struct {
	mu    sync.Mutex
	rules []filter.Rule
}

func (r *ruleRecorder) record(rule filter.Rule) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rules = append(r.rules, rule)
}

func (r *ruleRecorder) delivered() []filter.Rule {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]filter.Rule(nil), r.rules...)
}

// newRuleManager returns a Manager built by NewManager over a temp cache dir
// and a fresh Remote, with a recorder registered for rule changes.
func newRuleManager(t *testing.T) (*Manager, *kbtest.Remote, *ruleRecorder) {
	t.Helper()
	t.Setenv("P2KB_CACHE_DIR", t.TempDir())
	r := kbtest.NewRemote(t)
	m := NewManager(r.Client())
	rec := &ruleRecorder{}
	m.OnRuleChange(rec.record)
	return m, r, rec
}

// serveBlock makes the Remote's index carry block (empty for none).
func serveBlock(r *kbtest.Remote, block string) {
	idx := r.Index()
	idx.DeliveryFilter = nil
	if block != "" {
		idx.DeliveryFilter = json.RawMessage(block)
	}
	r.SetIndex(idx)
}

// writeLastGood persists raw as the last-good rule file.
func writeLastGood(t *testing.T, m *Manager, raw string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(m.lastGoodPath()), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(m.lastGoodPath(), []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}
}

func assertStatus(t *testing.T, got FilterStatus, wantRule filter.Rule, wantSource string) {
	t.Helper()
	if got.Source != wantSource {
		t.Errorf("Source = %q, want %q", got.Source, wantSource)
	}
	if !reflect.DeepEqual(got.Rule, wantRule) {
		t.Errorf("Rule = %+v, want %+v", got.Rule, wantRule)
	}
	if got.RuleID != wantRule.RuleID() {
		t.Errorf("RuleID = %s, want %s", got.RuleID, wantRule.RuleID())
	}
}

func TestNewManagerStartsOnBuiltinRule(t *testing.T) {
	m, _, _ := newRuleManager(t)
	got := m.FilterStatus()
	assertStatus(t, got, filter.BuiltinRule, RuleSourceBuiltIn)
	if got.RuleID != "b431af2a9f515c11fe5fd982642c636c6f8843bf89ed2c3003ee0e28c04877ee" {
		t.Errorf("built-in RuleID = %s", got.RuleID)
	}
}

func TestValidBlockIsRuleInEffectAndLastGood(t *testing.T) {
	m, r, rec := newRuleManager(t)
	serveBlock(r, indexRuleBlock)

	if err := m.EnsureIndex(); err != nil {
		t.Fatalf("EnsureIndex: %v", err)
	}
	got := m.FilterStatus()
	assertStatus(t, got, indexRule, RuleSourceIndex)
	if got.Refused != nil {
		t.Errorf("Refused = %+v, want nil", got.Refused)
	}

	data, err := os.ReadFile(m.lastGoodPath())
	if err != nil {
		t.Fatalf("last-good file not written: %v", err)
	}
	saved, err := filter.ParseRule(data)
	if err != nil || !reflect.DeepEqual(saved, indexRule) {
		t.Errorf("last-good file = %s (%v), want the index rule", data, err)
	}

	if d := rec.delivered(); len(d) != 1 || !reflect.DeepEqual(d[0], indexRule) {
		t.Errorf("callback deliveries = %+v, want exactly the index rule", d)
	}
}

func TestAbsentBlockFallsToLastGood(t *testing.T) {
	m, _, rec := newRuleManager(t)
	writeLastGood(t, m, indexRuleBlock)

	if err := m.EnsureIndex(); err != nil {
		t.Fatalf("EnsureIndex: %v", err)
	}
	got := m.FilterStatus()
	assertStatus(t, got, indexRule, RuleSourceLastGood)
	if got.Refused != nil {
		t.Errorf("Refused = %+v, want nil for an absent block", got.Refused)
	}
	if d := rec.delivered(); len(d) != 1 || !reflect.DeepEqual(d[0], indexRule) {
		t.Errorf("callback deliveries = %+v, want the last-good rule once", d)
	}
}

func TestAbsentBlockWithoutLastGoodIsBuiltin(t *testing.T) {
	m, _, rec := newRuleManager(t)
	if err := m.EnsureIndex(); err != nil {
		t.Fatalf("EnsureIndex: %v", err)
	}
	assertStatus(t, m.FilterStatus(), filter.BuiltinRule, RuleSourceBuiltIn)
	if _, err := os.Stat(m.lastGoodPath()); !os.IsNotExist(err) {
		t.Errorf("built-in rule must not be persisted as last-good (stat err %v)", err)
	}
	if d := rec.delivered(); len(d) != 1 {
		t.Errorf("callback deliveries = %d, want 1 (first resolution always delivers)", len(d))
	}
}

func TestRefusedBlockFallsToLastGoodAndIsReported(t *testing.T) {
	m, r, _ := newRuleManager(t)
	writeLastGood(t, m, indexRuleBlock)
	serveBlock(r, format2Block)

	if err := m.EnsureIndex(); err != nil {
		t.Fatalf("EnsureIndex: %v", err)
	}
	got := m.FilterStatus()
	assertStatus(t, got, indexRule, RuleSourceLastGood)
	if got.Refused == nil || got.Refused.Format != 2 || got.Refused.Reason == "" {
		t.Errorf("Refused = %+v, want format 2 with a reason", got.Refused)
	}
	data, _ := os.ReadFile(m.lastGoodPath())
	if saved, err := filter.ParseRule(data); err != nil || !reflect.DeepEqual(saved, indexRule) {
		t.Errorf("a refused block overwrote last-good: %s", data)
	}
}

func TestRefusedBlockWithoutLastGoodIsBuiltinAndReported(t *testing.T) {
	m, r, _ := newRuleManager(t)
	serveBlock(r, format2Block)

	if err := m.EnsureIndex(); err != nil {
		t.Fatalf("EnsureIndex: %v", err)
	}
	got := m.FilterStatus()
	assertStatus(t, got, filter.BuiltinRule, RuleSourceBuiltIn)
	if got.Refused == nil || got.Refused.Format != 2 {
		t.Errorf("Refused = %+v, want format 2", got.Refused)
	}
}

func TestValidBlockClearsEarlierRefusal(t *testing.T) {
	m, r, _ := newRuleManager(t)
	serveBlock(r, format2Block)
	if err := m.EnsureIndex(); err != nil {
		t.Fatalf("EnsureIndex: %v", err)
	}
	serveBlock(r, indexRuleBlock)
	if err := m.Refresh(); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	got := m.FilterStatus()
	assertStatus(t, got, indexRule, RuleSourceIndex)
	if got.Refused != nil {
		t.Errorf("Refused = %+v, want nil once a valid block arrives", got.Refused)
	}
}

func TestCorruptLastGoodIsBuiltinNeverEmpty(t *testing.T) {
	for name, raw := range map[string]string{
		"not json":      "{{{",
		"empty list":    `{"format":1,"remove_fields":[],"remove_column0_comments":true}`,
		"missing field": `{"format":1,"remove_fields":["source"]}`,
		"empty file":    "",
	} {
		t.Run(name, func(t *testing.T) {
			m, _, _ := newRuleManager(t)
			writeLastGood(t, m, raw)
			if err := m.EnsureIndex(); err != nil {
				t.Fatalf("EnsureIndex: %v", err)
			}
			got := m.FilterStatus()
			assertStatus(t, got, filter.BuiltinRule, RuleSourceBuiltIn)
			if len(got.Rule.RemoveFields) == 0 {
				t.Error("rule in effect is empty")
			}
		})
	}
}

func TestStartupResolvesRuleFromStaleCachedIndex(t *testing.T) {
	m, r, rec := newRuleManager(t)
	cached := `{"system":{"version":"old"},"files":{},"delivery_filter":` + indexRuleBlock + `}`
	if err := m.saveToCache([]byte(cached)); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(m.indexPath, old, old); err != nil {
		t.Fatal(err)
	}

	m.ResolveStartupRule()

	assertStatus(t, m.FilterStatus(), indexRule, RuleSourceIndex)
	if d := rec.delivered(); len(d) != 1 || !reflect.DeepEqual(d[0], indexRule) {
		t.Errorf("callback deliveries = %+v, want the cached index's rule", d)
	}
	if got := r.Hits(fetch.IndexPath); got != 0 {
		t.Errorf("startup resolution fetched the index %d times, want 0", got)
	}
	m.mu.RLock()
	installed := m.index
	m.mu.RUnlock()
	if installed != nil {
		t.Error("startup resolution installed the stale index")
	}
}

func TestStartupWithoutCachedIndexUsesLastGood(t *testing.T) {
	m, _, rec := newRuleManager(t)
	writeLastGood(t, m, indexRuleBlock)
	m.ResolveStartupRule()
	assertStatus(t, m.FilterStatus(), indexRule, RuleSourceLastGood)
	if len(rec.delivered()) != 1 {
		t.Errorf("callback deliveries = %d, want 1", len(rec.delivered()))
	}
}

func TestFreshCachedIndexResolvesRuleWithoutFetch(t *testing.T) {
	m, r, _ := newRuleManager(t)
	cached := `{"system":{"version":"c"},"files":{},"delivery_filter":` + indexRuleBlock + `}`
	if err := m.saveToCache([]byte(cached)); err != nil {
		t.Fatal(err)
	}
	if err := m.EnsureIndex(); err != nil {
		t.Fatalf("EnsureIndex: %v", err)
	}
	assertStatus(t, m.FilterStatus(), indexRule, RuleSourceIndex)
	if got := r.Hits(fetch.IndexPath); got != 0 {
		t.Errorf("index fetches = %d, want 0 (cache was fresh)", got)
	}
}

// TestCallbackFiresOnlyWhenRuleChanges defines the delivery contract: the
// first resolution always delivers; a re-resolution to the same rule_id does
// not; a different rule_id does.
func TestCallbackFiresOnlyWhenRuleChanges(t *testing.T) {
	m, r, rec := newRuleManager(t)
	serveBlock(r, indexRuleBlock)

	m.ResolveStartupRule() // no cached index, no last-good: built-in
	if err := m.EnsureIndex(); err != nil {
		t.Fatalf("EnsureIndex: %v", err)
	}
	if err := m.Refresh(); err != nil { // same block again
		t.Fatalf("Refresh: %v", err)
	}
	serveBlock(r, otherRuleBlock)
	if err := m.Refresh(); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	serveBlock(r, "") // block disappears: last-good is the rule just seen
	if err := m.Refresh(); err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	other, _ := filter.ParseRule(json.RawMessage(otherRuleBlock))
	want := []filter.Rule{filter.BuiltinRule, indexRule, other}
	if got := rec.delivered(); !reflect.DeepEqual(got, want) {
		t.Errorf("deliveries =\n  %+v\nwant\n  %+v", got, want)
	}
	assertStatus(t, m.FilterStatus(), other, RuleSourceLastGood)
}

// TestCallbackRunsWithoutDataLock proves the callback is not called under
// m.mu: a callback that reads FilterStatus would deadlock if it were.
func TestCallbackRunsWithoutDataLock(t *testing.T) {
	m, r, _ := newRuleManager(t)
	serveBlock(r, indexRuleBlock)
	seen := make(chan FilterStatus, 4)
	m.OnRuleChange(func(filter.Rule) { seen <- m.FilterStatus() })

	done := make(chan error, 1)
	go func() { done <- m.EnsureIndex() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("EnsureIndex: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("EnsureIndex deadlocked: the callback ran under the data lock")
	}
	got := <-seen
	if got.Source != RuleSourceIndex {
		t.Errorf("status seen inside the callback = %+v, want the new rule already recorded", got)
	}
}

func TestConcurrentInstallsAndStatusReads(t *testing.T) {
	m, r, rec := newRuleManager(t)
	serveBlock(r, indexRuleBlock)
	m.ttl = time.Nanosecond // every EnsureIndex takes the slow path

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(3)
		go func() { defer wg.Done(); _ = m.EnsureIndex() }()
		go func() { defer wg.Done(); _ = m.Refresh() }()
		go func() {
			defer wg.Done()
			if s := m.FilterStatus(); len(s.Rule.RemoveFields) == 0 {
				t.Error("FilterStatus returned an empty rule")
			}
		}()
	}
	wg.Wait()

	assertStatus(t, m.FilterStatus(), indexRule, RuleSourceIndex)
	if d := rec.delivered(); len(d) != 1 {
		t.Errorf("deliveries = %d, want 1 (same rule every install)", len(d))
	}
}
