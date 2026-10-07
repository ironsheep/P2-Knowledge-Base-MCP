package filter

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

// builtinRuleID is the agreement's published rule_id for BuiltinRule (§3.1).
const builtinRuleID = "b431af2a9f515c11fe5fd982642c636c6f8843bf89ed2c3003ee0e28c04877ee"

func TestBuiltinRuleID(t *testing.T) {
	if got := BuiltinRule.RuleID(); got != builtinRuleID {
		t.Errorf("BuiltinRule.RuleID() = %s, want %s", got, builtinRuleID)
	}
	if err := BuiltinRule.Validate(); err != nil {
		t.Errorf("BuiltinRule.Validate() = %v, want nil", err)
	}
}

func TestParseRuleAcceptsBuiltinBlock(t *testing.T) {
	raw := `{
  "format": 1,
  "remove_fields": [
    "documentation_level", "documentation_source", "enhancement_source", "last_updated",
    "manual_extraction_date", "source", "source_reference", "sources", "verified_against"
  ],
  "remove_column0_comments": true
}`
	r, err := ParseRule(json.RawMessage(raw))
	if err != nil {
		t.Fatalf("ParseRule() error: %v", err)
	}
	if !reflect.DeepEqual(r, BuiltinRule) {
		t.Errorf("ParseRule() = %+v, want BuiltinRule", r)
	}
	if got := r.RuleID(); got != builtinRuleID {
		t.Errorf("RuleID() = %s, want %s", got, builtinRuleID)
	}
}

// TestParseRuleRuleIDMatchesTextForm proves the id is taken over the fields in
// the order the index lists them, with the boolean spelled out.
func TestParseRuleRuleIDMatchesTextForm(t *testing.T) {
	r, err := ParseRule(json.RawMessage(`{"format":1,"remove_fields":["zeta","alpha"],"remove_column0_comments":false}`))
	if err != nil {
		t.Fatalf("ParseRule() error: %v", err)
	}
	sum := sha256.Sum256([]byte("p2kb-delivery-filter\nformat=1\nremove_fields=zeta,alpha\nremove_column0_comments=false\n"))
	if got, want := r.RuleID(), hex.EncodeToString(sum[:]); got != want {
		t.Errorf("RuleID() = %s, want %s", got, want)
	}
}

func TestParseRuleRefuses(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantFormat int
	}{
		{"missing remove_column0_comments", `{"format":1,"remove_fields":["source"]}`, 1},
		{"missing remove_fields", `{"format":1,"remove_column0_comments":true}`, 1},
		{"format is a string", `{"format":"1","remove_fields":["source"],"remove_column0_comments":true}`, 0},
		{"format missing", `{"remove_fields":["source"],"remove_column0_comments":true}`, 0},
		{"boolean is a string", `{"format":1,"remove_fields":["source"],"remove_column0_comments":"true"}`, 1},
		{"extra key", `{"format":1,"remove_fields":["source"],"remove_column0_comments":true,"extra":1}`, 1},
		{"duplicate field", `{"format":1,"remove_fields":["source","source"],"remove_column0_comments":true}`, 1},
		{"empty list", `{"format":1,"remove_fields":[],"remove_column0_comments":true}`, 1},
		{"name contains '-'", `{"format":1,"remove_fields":["last-updated"],"remove_column0_comments":true}`, 1},
		{"format 2", `{"format":2,"remove_fields":["source"],"remove_column0_comments":true}`, 2},
		{"not an object", `[1]`, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := ParseRule(json.RawMessage(tt.raw))
			var refused *RefusedRule
			if !errors.As(err, &refused) {
				t.Fatalf("ParseRule() = %+v, %v; want *RefusedRule", r, err)
			}
			if refused.Format != tt.wantFormat {
				t.Errorf("RefusedRule.Format = %d, want %d", refused.Format, tt.wantFormat)
			}
		})
	}
}

// applyCases is the engine table (agreement §4). Every case but the empty one
// produces output the replaced five-field regex filter would not.
var applyCases = []struct {
	name  string
	rule  Rule
	input string
	want  string
}{
	{
		name:  "empty input gives empty output",
		rule:  BuiltinRule,
		input: "",
		want:  "",
	},
	{
		name:  "missing trailing newline is added",
		rule:  BuiltinRule,
		input: "mnemonic: MOV",
		want:  "mnemonic: MOV\n",
	},
	{
		name:  "space before the colon is not a field",
		rule:  BuiltinRule,
		input: "source : kept\nsource: removed\n",
		want:  "source : kept\n",
	},
	{
		name:  "longer name is not a field",
		rule:  BuiltinRule,
		input: "sources_x: kept\nsources: removed\n",
		want:  "sources_x: kept\n",
	},
	{
		name:  "list item and flow mapping keys are content",
		rule:  BuiltinRule,
		input: "examples:\n  - source: \"CON { Motor Constants }\"\nREG_ARRAY: {source: \"cog registers\"}\nverified_against: removed\n",
		want:  "examples:\n  - source: \"CON { Motor Constants }\"\nREG_ARRAY: {source: \"cog registers\"}\n",
	},
	{
		name:  "field line is removed with its value",
		rule:  BuiltinRule,
		input: "a: 1\nsource: \"x\"\nb: 2\n",
		want:  "a: 1\nb: 2\n",
	},
	{
		name:  "tab counts as one indent byte",
		rule:  BuiltinRule,
		input: "m:\n\tsources:\n  - deeper than one byte\n\tz: 1\n",
		want:  "m:\n\tz: 1\n",
	},
	{
		name:  "leading carriage return is whitespace",
		rule:  BuiltinRule,
		input: "a:\r\n  source: x\n\r  more of the span\n  b: 1\n\rsources: y\n",
		want:  "a:\r\n  b: 1\n",
	},
	{
		name:  "span ends at a same-indent sibling",
		rule:  BuiltinRule,
		input: "sources:\n  - a\n  - b\nnext: 1\n",
		want:  "next: 1\n",
	},
	{
		name:  "span opened at depth",
		rule:  BuiltinRule,
		input: "item:\n  details:\n    sources:\n      - a\n    kept: 1\n  also: 2\n",
		want:  "item:\n  details:\n    kept: 1\n  also: 2\n",
	},
	{
		name:  "blank line inside a span is removed, outside is kept",
		rule:  BuiltinRule,
		input: "a: 1\n\nsource: >-\n  line one\n\n  line two\nnext: 1\n",
		want:  "a: 1\n\nnext: 1\n",
	},
	{
		name:  "comments inside a span do not end it",
		rule:  BuiltinRule,
		input: "sources:\n  - ledger B-112\n# note\n  - engineering/foo.txt\n    # indented note\n  - more\nnext: 1\n",
		want:  "next: 1\n",
	},
	{
		name:  "comments inside a span are removed without the column-0 rule",
		rule:  Rule{Format: 1, RemoveFields: []string{"sources"}, RemoveColumn0Comments: false},
		input: "sources:\n  - ledger B-112\n# note\n  - engineering/foo.txt\nnext: 1\n",
		want:  "next: 1\n",
	},
	{
		name:  "column-0 comment removed when the rule says so",
		rule:  BuiltinRule,
		input: "# maintainer note\na: 1\n",
		want:  "a: 1\n",
	},
	{
		name:  "column-0 comment kept when the rule says not",
		rule:  Rule{Format: 1, RemoveFields: []string{"verified_against"}, RemoveColumn0Comments: false},
		input: "# maintainer note\nverified_against: x\nb: 1\n",
		want:  "# maintainer note\nb: 1\n",
	},
	{
		name:  "indented comment outside a span is kept",
		rule:  BuiltinRule,
		input: "a:\n  # note\n  b: 1\nsource: x\n",
		want:  "a:\n  # note\n  b: 1\n",
	},
}

func TestApply(t *testing.T) {
	for _, tt := range applyCases {
		t.Run(tt.name, func(t *testing.T) {
			if got := Apply(tt.rule, tt.input); got != tt.want {
				t.Errorf("Apply()\n  input %q\n  got   %q\n  want  %q", tt.input, got, tt.want)
			}
		})
	}
}
