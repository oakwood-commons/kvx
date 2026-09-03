package formatter

import (
	"reflect"

	"github.com/pelletier/go-toml/v2"
)

// FormatTOML renders an object to TOML. TOML documents must be tables at the
// root, so slices/arrays are wrapped in map[string]any{"items": data} and
// scalars (strings, numbers, bools, etc.) are wrapped in
// map[string]any{"value": data} before marshaling.
func FormatTOML(v any) (string, error) {
	if v == nil {
		return "", nil
	}

	// Treat typed-nil pointers/interfaces as nil input.
	rv := reflect.ValueOf(v)
	if (rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface) && rv.IsNil() {
		return "", nil
	}

	data := v
	switch {
	case isSliceOrArrayKind(v):
		data = map[string]any{"items": deref(v)}
	case !isTableRootKind(v):
		data = map[string]any{"value": deref(v)}
	}

	b, err := toml.Marshal(data)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

// isTableRootKind reports whether v can be marshaled directly as a TOML
// document root (a map or struct). It unwraps pointers and interfaces before
// checking the final kind.
func isTableRootKind(v any) bool {
	if v == nil {
		return false
	}

	rv := reflect.ValueOf(v)
	for rv.IsValid() && (rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface) {
		if rv.IsNil() {
			return false
		}
		rv = rv.Elem()
	}

	return rv.IsValid() && (rv.Kind() == reflect.Map || rv.Kind() == reflect.Struct)
}

// isSliceOrArrayKind reports whether v is a slice or array type.
// It unwraps pointers and interfaces before checking the final kind.
func isSliceOrArrayKind(v any) bool {
	if v == nil {
		return false
	}

	rv := reflect.ValueOf(v)
	for rv.IsValid() && (rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface) {
		if rv.IsNil() {
			return false
		}
		rv = rv.Elem()
	}

	return rv.IsValid() && (rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array)
}

// deref unwraps pointers and interfaces to return the underlying value.
func deref(v any) any {
	rv := reflect.ValueOf(v)
	for rv.IsValid() && (rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Interface) {
		if rv.IsNil() {
			return v
		}
		rv = rv.Elem()
	}

	if !rv.IsValid() {
		return v
	}

	return rv.Interface()
}
