package limiter

import (
	"fmt"
	"reflect"
	"sort"
)

// Config holds the record-limiting parameters.
type Config struct {
	Limit  int // Show only this many records (0 = unlimited)
	Offset int // Skip the first N records (0 = no skip)
	Tail   int // Show only the last N records (0 = disabled); mutually exclusive with Limit
}

// Validate checks for conflicting flag combinations and returns an error if invalid.
// Rules:
// - Limit and Tail are mutually exclusive
// - If Tail is set, Offset is ignored
// - All numeric values must be non-negative
func (c Config) Validate() error {
	if c.Limit < 0 {
		return fmt.Errorf("--limit must be non-negative, got %d", c.Limit)
	}
	if c.Offset < 0 {
		return fmt.Errorf("--offset must be non-negative, got %d", c.Offset)
	}
	if c.Tail < 0 {
		return fmt.Errorf("--tail must be non-negative, got %d", c.Tail)
	}

	// Check for mutually exclusive flags
	if c.Limit > 0 && c.Tail > 0 {
		return fmt.Errorf("--limit and --tail are mutually exclusive")
	}

	return nil
}

// IsActive returns true if any limiting is configured.
func (c Config) IsActive() bool {
	return c.Limit > 0 || c.Offset > 0 || c.Tail > 0
}

// Apply applies the limiting configuration to the given data.
// Handles arrays and maps; returns the limited subset.
// For maps, uses stable key ordering (sorted keys).
func (c Config) Apply(data interface{}) interface{} {
	if !c.IsActive() {
		return data
	}

	switch v := data.(type) {
	case []interface{}:
		return c.applyToArray(v)
	case map[string]interface{}:
		return c.applyToMap(v)
	default:
		// For scalar values or unknown types, return unchanged
		return data
	}
}

// applyToArray applies limiting to an array.
func (c Config) applyToArray(arr []interface{}) interface{} {
	length := len(arr)

	// Handle --tail (show last N records)
	if c.Tail > 0 {
		start := length - c.Tail
		if start < 0 {
			start = 0
		}
		return arr[start:]
	}

	// Handle --offset and --limit
	start := c.Offset
	if start > length {
		start = length
	}

	var end int
	if c.Limit > 0 {
		end = start + c.Limit
		if end > length {
			end = length
		}
	} else {
		end = length
	}

	if start > end {
		start = end
	}

	return arr[start:end]
}

// applyToMap applies limiting to a map by converting to a sorted list of key-value pairs,
// then limiting, then converting back to a map.
func (c Config) applyToMap(m map[string]interface{}) interface{} {
	// Sort keys for stable, deterministic ordering
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Handle --tail (show last N records)
	if c.Tail > 0 {
		start := len(keys) - c.Tail
		if start < 0 {
			start = 0
		}
		keys = keys[start:]
	} else {
		// Handle --offset and --limit
		start := c.Offset
		if start > len(keys) {
			start = len(keys)
		}

		var end int
		if c.Limit > 0 {
			end = start + c.Limit
			if end > len(keys) {
				end = len(keys)
			}
		} else {
			end = len(keys)
		}

		if start > end {
			start = end
		}

		keys = keys[start:end]
	}

	// Reconstruct map with limited keys
	result := make(map[string]interface{})
	for _, k := range keys {
		result[k] = m[k]
	}
	return result
}

// ApplyToSliceOfMaps applies limiting to a slice of maps.
// Used when the result of an expression is []map[string]interface{} (array of objects).
func (c Config) ApplyToSliceOfMaps(data interface{}) interface{} {
	arr, ok := data.([]interface{})
	if !ok {
		return data
	}

	// Apply limiting to the array itself
	return c.applyToArray(arr)
}

// ApplyToGenericSlice applies limiting to a generic slice type.
// Uses reflection to handle various slice types.
func (c Config) ApplyToGenericSlice(data interface{}) interface{} {
	val := reflect.ValueOf(data)
	if val.Kind() != reflect.Slice {
		return data
	}

	length := val.Len()

	// Handle --tail (show last N records)
	var start, end int
	if c.Tail > 0 {
		start = length - c.Tail
		if start < 0 {
			start = 0
		}
		end = length
	} else {
		// Handle --offset and --limit
		start = c.Offset
		if start > length {
			start = length
		}

		if c.Limit > 0 {
			end = start + c.Limit
			if end > length {
				end = length
			}
		} else {
			end = length
		}

		if start > end {
			start = end
		}
	}

	// Use reflection to slice
	return val.Slice(start, end).Interface()
}
