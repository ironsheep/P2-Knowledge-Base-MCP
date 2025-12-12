// Package filter provides content filtering for P2KB YAML files.
package filter

import (
	"regexp"
	"strings"
)

// metadataPattern matches metadata lines that should be removed.
// These lines are internal tracking data that wastes tokens.
var metadataPattern = regexp.MustCompile(
	`(?m)^\s*(last_updated|enhancement_source|documentation_source|documentation_level|manual_extraction_date):.*\n?`)

// FilterMetadata removes internal metadata lines from YAML content.
// This saves tokens by removing tracking data that's not useful for AI consumption.
//
// Filtered fields:
//   - last_updated
//   - enhancement_source
//   - documentation_source
//   - documentation_level
//   - manual_extraction_date
func FilterMetadata(content string) string {
	return metadataPattern.ReplaceAllString(content, "")
}

// FilterMetadataLines removes metadata lines and returns the result.
// This is an alternative implementation that processes line by line.
func FilterMetadataLines(content string) string {
	lines := strings.Split(content, "\n")
	var filtered []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if shouldFilterLine(trimmed) {
			continue
		}
		filtered = append(filtered, line)
	}

	return strings.Join(filtered, "\n")
}

// shouldFilterLine returns true if the line should be filtered out.
func shouldFilterLine(line string) bool {
	prefixes := []string{
		"last_updated:",
		"enhancement_source:",
		"documentation_source:",
		"documentation_level:",
		"manual_extraction_date:",
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}

	return false
}

// CountFilteredLines counts how many lines would be filtered.
func CountFilteredLines(content string) int {
	lines := strings.Split(content, "\n")
	count := 0

	for _, line := range lines {
		if shouldFilterLine(strings.TrimSpace(line)) {
			count++
		}
	}

	return count
}
