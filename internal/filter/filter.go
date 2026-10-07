// Package filter removes provenance from P2KB YAML files before delivery.
//
// The rule and the engine are defined by the delivery-filter interface
// agreement between the P2KB and p2kb-mcp (format 1). The code below is the
// agreement's Go reference implementation, carried verbatim: certification
// compares this engine's output byte-for-byte with the KB's reference engine,
// so do not "improve" it.
package filter

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// EngineVersion identifies the behaviour of Apply and ParseRule. It is the
// engine half of the cache stamp (the rule_id is the other half), so bump it
// whenever Apply or ParseRule changes behaviour; every cache filtered by the
// old engine is then discarded.
const EngineVersion = "1"

// Rule is a delivery_filter block: which fields to remove and whether to
// remove column-0 comments.
type Rule struct {
	Format                int
	RemoveFields          []string
	RemoveColumn0Comments bool
}

// BuiltinRule is the §3 block exactly. rule_id b431af2a…77ee (§3.1).
var BuiltinRule = Rule{
	Format: 1,
	RemoveFields: []string{
		"documentation_level", "documentation_source", "enhancement_source", "last_updated",
		"manual_extraction_date", "source", "source_reference", "sources", "verified_against",
	},
	RemoveColumn0Comments: true,
}

// posixSpace is POSIX [[:space:]].
const posixSpace = " \t\n\v\f\r"

// fieldNameRe matches a literal key name; remove_fields holds no patterns.
var fieldNameRe = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// RefusedRule is returned when the index carries a block this engine must not apply.
// Format is the block's format when it could be read, else 0.
type RefusedRule struct {
	Format int
	Reason string
}

func (e *RefusedRule) Error() string { return "delivery_filter refused: " + e.Reason }

// ParseRule reads the index's "delivery_filter" value. Strict: every key required, exact types,
// no extra keys, no duplicate or invalid names. Anything else is a *RefusedRule (§3, §5).
func ParseRule(raw json.RawMessage) (Rule, error) {
	var head struct {
		Format *int `json:"format"`
	}
	if err := json.Unmarshal(raw, &head); err != nil || head.Format == nil {
		return Rule{}, &RefusedRule{Reason: "format missing or not an integer"}
	}
	if *head.Format != 1 {
		return Rule{}, &RefusedRule{Format: *head.Format, Reason: fmt.Sprintf("unsupported format %d", *head.Format)}
	}
	var p struct {
		Format                *int      `json:"format"`
		RemoveFields          *[]string `json:"remove_fields"`
		RemoveColumn0Comments *bool     `json:"remove_column0_comments"`
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return Rule{}, &RefusedRule{Format: 1, Reason: err.Error()}
	}
	if p.RemoveFields == nil || p.RemoveColumn0Comments == nil {
		return Rule{}, &RefusedRule{Format: 1, Reason: "remove_fields or remove_column0_comments missing"}
	}
	r := Rule{Format: 1, RemoveFields: *p.RemoveFields, RemoveColumn0Comments: *p.RemoveColumn0Comments}
	if err := r.Validate(); err != nil {
		return Rule{}, &RefusedRule{Format: 1, Reason: err.Error()}
	}
	return r, nil
}

// Validate reports whether this engine can apply r. Format 1 only.
func (r Rule) Validate() error {
	if r.Format != 1 {
		return fmt.Errorf("unsupported format %d", r.Format)
	}
	if len(r.RemoveFields) == 0 {
		return fmt.Errorf("remove_fields is empty")
	}
	seen := map[string]bool{}
	for _, f := range r.RemoveFields {
		if !fieldNameRe.MatchString(f) {
			return fmt.Errorf("field %q is not a literal key name", f)
		}
		if seen[f] {
			return fmt.Errorf("field %q listed twice", f)
		}
		seen[f] = true
	}
	return nil
}

// RuleID is §3.1: SHA-256 of the canonical text, lowercase hex.
func (r Rule) RuleID() string {
	text := fmt.Sprintf("p2kb-delivery-filter\nformat=%d\nremove_fields=%s\nremove_column0_comments=%t\n",
		r.Format, strings.Join(r.RemoveFields, ","), r.RemoveColumn0Comments)
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// leadWS returns the count of leading whitespace bytes in s; a tab counts as 1.
func leadWS(s string) int { return len(s) - len(strings.TrimLeft(s, posixSpace)) }

// Apply removes provenance from one raw YAML file. Hash the raw body BEFORE calling this.
func Apply(r Rule, content string) string {
	if content == "" {
		return ""
	}
	fields := make(map[string]bool, len(r.RemoveFields))
	for _, f := range r.RemoveFields {
		fields[f] = true
	}
	lines := strings.Split(content, "\n")
	if strings.HasSuffix(content, "\n") {
		lines = lines[:len(lines)-1]
	}
	var b strings.Builder
	dropping, dropIndent := false, 0
	for _, line := range lines {
		if dropping {
			trimmed := strings.TrimLeft(line, posixSpace)
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				continue // blank or comment-only: removed, span continues
			}
			if leadWS(line) > dropIndent {
				continue
			}
			dropping = false // span ended; this line is still tested below
		}
		if r.RemoveColumn0Comments && strings.HasPrefix(line, "#") {
			continue
		}
		ind := leadWS(line)
		rest := line[ind:]
		if i := strings.IndexByte(rest, ':'); i > 0 && fields[rest[:i]] {
			dropIndent, dropping = ind, true
			continue
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}
