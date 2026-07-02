package formatter

import (
	"reflect"

	"github.com/pelletier/go-toml/v2"
)

// FormatTOML renders an object to TOML. TOML does not support a bare array as
// the document root, so slices/arrays are wrapped in map[string]any{"items": data}
// before marshaling.
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
	if isSliceOrArrayKind(v) {
		data = map[string]any{"items": deref(v)}
	}

	b, err := toml.Marshal(data)
	if err != nil {
		return "", err
	}

	return string(b), nil
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
