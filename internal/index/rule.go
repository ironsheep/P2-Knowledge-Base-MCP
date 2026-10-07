package index

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/ironsheep/p2kb-mcp/internal/filter"
	"github.com/ironsheep/p2kb-mcp/internal/logging"
)

// Rule sources, as p2kb_version reports them (agreement §5, §7).
const (
	RuleSourceIndex    = "index"
	RuleSourceLastGood = "last-good"
	RuleSourceBuiltIn  = "built-in"
)

// lastGoodFile is the persisted last-good rule, next to the cached index.
const lastGoodFile = "delivery-filter.json"

// FilterStatus is the delivery-filter rule in effect and how it was determined.
type FilterStatus struct {
	Rule   filter.Rule
	RuleID string
	Source string
	// Refused is the index's delivery_filter block when it was refused, else nil.
	Refused *filter.RefusedRule
}

// builtinStatus is the status before any rule has been resolved.
func builtinStatus() FilterStatus {
	return FilterStatus{Rule: filter.BuiltinRule, RuleID: filter.BuiltinRule.RuleID(), Source: RuleSourceBuiltIn}
}

// ruleBlock is the index's delivery_filter block, the form the last-good rule
// is persisted in so it is re-read through filter.ParseRule.
type ruleBlock struct {
	Format                int      `json:"format"`
	RemoveFields          []string `json:"remove_fields"`
	RemoveColumn0Comments bool     `json:"remove_column0_comments"`
}

// OnRuleChange registers fn to receive the rule in effect whenever it changes,
// starting with the first rule resolved. fn is called with no index lock held,
// in resolution order; it must not call back into the index manager.
func (m *Manager) OnRuleChange(fn func(filter.Rule)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onRule = fn
}

// FilterStatus returns the rule in effect and its source.
func (m *Manager) FilterStatus() FilterStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.filterStatus
}

// ResolveStartupRule determines the rule in effect before the first fetch,
// from the cached index file whatever its age (agreement §5: "at startup,
// against the cached index"), and delivers it. It installs no index: a stale
// cached index is still refreshed on first use.
func (m *Manager) ResolveStartupRule() {
	m.fetchMu.Lock()
	defer m.fetchMu.Unlock()

	var raw json.RawMessage
	if data, err := os.ReadFile(m.indexPath); err == nil {
		var head struct {
			DeliveryFilter json.RawMessage `json:"delivery_filter"`
		}
		if json.Unmarshal(data, &head) == nil {
			raw = head.DeliveryFilter
		}
	}
	m.setFilterStatus(m.resolveRule(raw))
}

// installIndex makes idx the index in effect, refreshed at refreshed, and
// resolves the rule from it. data, when non-nil, is written to the index cache
// file. The caller holds fetchMu, which serializes installs, the last-good
// file and rule delivery; m.mu is taken only to swap state, so no file I/O and
// no callback happens under it.
func (m *Manager) installIndex(idx *Index, data []byte, refreshed time.Time) {
	status := m.resolveRule(idx.DeliveryFilter)
	if data != nil {
		if err := m.saveToCache(data); err != nil {
			logging.Warnf("p2kb-mcp: warning: failed to cache index: %v", err)
		}
	}

	m.mu.Lock()
	m.index = idx
	m.lastRefresh = refreshed
	m.mu.Unlock()

	m.setFilterStatus(status)
}

// setFilterStatus records status and delivers its rule if the rule changed.
// The caller holds fetchMu.
func (m *Manager) setFilterStatus(status FilterStatus) {
	m.mu.Lock()
	m.filterStatus = status
	fn := m.onRule
	m.mu.Unlock()

	if fn != nil && status.RuleID != m.deliveredRuleID {
		m.deliveredRuleID = status.RuleID
		fn(status.Rule)
	}
}

// resolveRule is agreement §5. A block ParseRule accepts is the rule, and is
// persisted as last-good. A missing or refused block falls to the last-good
// rule, else BuiltinRule, so the server never filters less than it last did;
// a refused block is reported. The caller holds fetchMu.
func (m *Manager) resolveRule(raw json.RawMessage) FilterStatus {
	var refused *filter.RefusedRule
	if len(raw) > 0 {
		rule, err := filter.ParseRule(raw)
		if err == nil {
			if err := m.saveLastGood(rule); err != nil {
				logging.Warnf("p2kb-mcp: warning: failed to save last-good filter rule: %v", err)
			}
			return FilterStatus{Rule: rule, RuleID: rule.RuleID(), Source: RuleSourceIndex}
		}
		if !errors.As(err, &refused) {
			refused = &filter.RefusedRule{Reason: err.Error()}
		}
	}

	status := builtinStatus()
	if rule, ok := m.loadLastGood(); ok {
		status = FilterStatus{Rule: rule, RuleID: rule.RuleID(), Source: RuleSourceLastGood}
	}
	status.Refused = refused
	return status
}

func (m *Manager) lastGoodPath() string {
	return filepath.Join(filepath.Dir(m.indexPath), lastGoodFile)
}

// loadLastGood reads the persisted rule. A missing, unreadable or refused
// file is no rule at all.
func (m *Manager) loadLastGood() (filter.Rule, bool) {
	data, err := os.ReadFile(m.lastGoodPath())
	if err != nil {
		return filter.Rule{}, false
	}
	rule, err := filter.ParseRule(data)
	if err != nil {
		logging.Warnf("p2kb-mcp: warning: ignoring last-good filter rule %s: %v", m.lastGoodPath(), err)
		return filter.Rule{}, false
	}
	return rule, true
}

// saveLastGood persists rule, replacing the file atomically so a crash never
// leaves a partial rule behind.
func (m *Manager) saveLastGood(rule filter.Rule) error {
	data, err := json.Marshal(ruleBlock{
		Format:                rule.Format,
		RemoveFields:          rule.RemoveFields,
		RemoveColumn0Comments: rule.RemoveColumn0Comments,
	})
	if err != nil {
		return err
	}
	path := m.lastGoodPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
