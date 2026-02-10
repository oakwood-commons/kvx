package navigator

import (
	"regexp"
)

// NormalizePath converts a dotted path to CEL notation.
// Examples:
//
//	"items.0" -> "items[0]"
//	"regions.asia.countries.1" -> "regions.asia.countries[1]"
//	"items.0.tags" -> "items[0].tags"
func NormalizePath(path string) string {
	if path == "" {
		return path
	}

	// Replace numeric segments preceded by a dot with bracket notation
	// Pattern: .(\d+) becomes [(\d+)]
	re := regexp.MustCompile(`\.(\d+)`)
	normalized := re.ReplaceAllString(path, "[$1]")

	return normalized
}
