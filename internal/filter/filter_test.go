package filter

import (
	"strings"
	"testing"
)

func TestFilterMetadata(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty content",
			input:    "",
			expected: "",
		},
		{
			name:     "no metadata",
			input:    "mnemonic: MOV\ndescription: Move data\n",
			expected: "mnemonic: MOV\ndescription: Move data\n",
		},
		{
			name: "single metadata field",
			input: `mnemonic: MOV
last_updated: "2025-01-01"
description: Move data
`,
			expected: `mnemonic: MOV
description: Move data
`,
		},
		{
			name: "multiple metadata fields",
			input: `mnemonic: MOV
last_updated: "2025-01-01"
enhancement_source: "test"
documentation_source: "manual"
documentation_level: "complete"
manual_extraction_date: "2025-01-01"
description: Move data
`,
			expected: `mnemonic: MOV
description: Move data
`,
		},
		{
			name: "metadata with spaces",
			input: `mnemonic: MOV
  last_updated: "2025-01-01"
description: Move data
`,
			expected: `mnemonic: MOV
description: Move data
`,
		},
		{
			name: "real-world example",
			input: `mnemonic: MOV
category: Data Movement
last_updated: "2025-12-12T10:30:00"
documentation_source: "P2 Manual v35"
syntax:
  - "MOV D,S"
description: |
  Copy source to destination.
related_instructions:
  - p2kbPasm2Add
  - p2kbPasm2Loc
`,
			expected: `mnemonic: MOV
category: Data Movement
syntax:
  - "MOV D,S"
description: |
  Copy source to destination.
related_instructions:
  - p2kbPasm2Add
  - p2kbPasm2Loc
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterMetadata(tt.input)
			if result != tt.expected {
				t.Errorf("FilterMetadata() =\n%q\nwant\n%q", result, tt.expected)
			}
		})
	}
}

func TestFilterMetadataLines(t *testing.T) {
	input := `mnemonic: MOV
last_updated: "2025-01-01"
description: Move data`

	result := FilterMetadataLines(input)
	if strings.Contains(result, "last_updated") {
		t.Error("FilterMetadataLines() should remove last_updated")
	}
	if !strings.Contains(result, "mnemonic: MOV") {
		t.Error("FilterMetadataLines() should preserve mnemonic")
	}
}

func TestCountFilteredLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "no metadata",
			input:    "mnemonic: MOV\n",
			expected: 0,
		},
		{
			name:     "one metadata field",
			input:    "last_updated: x\nmnemonic: MOV\n",
			expected: 1,
		},
		{
			name:     "all metadata fields",
			input:    "last_updated: x\nenhancement_source: y\ndocumentation_source: z\ndocumentation_level: a\nmanual_extraction_date: b\n",
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CountFilteredLines(tt.input)
			if result != tt.expected {
				t.Errorf("CountFilteredLines() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestShouldFilterLine(t *testing.T) {
	tests := []struct {
		line     string
		expected bool
	}{
		{"last_updated: x", true},
		{"enhancement_source: x", true},
		{"documentation_source: x", true},
		{"documentation_level: x", true},
		{"manual_extraction_date: x", true},
		{"mnemonic: MOV", false},
		{"description: test", false},
		{"", false},
		{"  last_updated: x", false}, // not trimmed - should be false
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			result := shouldFilterLine(tt.line)
			if result != tt.expected {
				t.Errorf("shouldFilterLine(%q) = %v, want %v", tt.line, result, tt.expected)
			}
		})
	}
}

func BenchmarkFilterMetadata(b *testing.B) {
	input := `mnemonic: MOV
category: Data Movement
last_updated: "2025-12-12T10:30:00"
documentation_source: "P2 Manual v35"
enhancement_source: "AI"
documentation_level: "complete"
manual_extraction_date: "2025-01-01"
syntax:
  - "MOV D,S"
description: |
  Copy source to destination.
related_instructions:
  - p2kbPasm2Add
  - p2kbPasm2Loc
`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FilterMetadata(input)
	}
}
