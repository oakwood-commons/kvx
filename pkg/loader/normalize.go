package loader

import (
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
)

var (
	jsonMarshalerType = reflect.TypeFor[json.Marshaler]()
	textMarshalerType = reflect.TypeFor[encoding.TextMarshaler]()
	isZeroerType      = reflect.TypeFor[interface{ IsZero() bool }]()
	structFieldsCache sync.Map // map[reflect.Type][]structFieldInfo
)

type visitKey struct {
	ptr uintptr
	typ reflect.Type
}

type inlineKind uint8

const (
	inlineNone inlineKind = iota
	inlineStruct
	inlineMap
)

type structFieldInfo struct {
	index     int
	key       string
	inline    inlineKind
	omitEmpty bool
	omitZero  bool
	asString  bool
	fromYAML  bool
}

// ValueNeedsNormalization reports whether value (or a nested element within a
// []any or map[string]any) contains Go types such as structs, pointers, typed
// slices/maps, nil byte slices, or named scalars that must be normalized before
// CEL evaluation.
func ValueNeedsNormalization(value any) bool {
	switch v := value.(type) {
	case nil, string, bool,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return false
	case []byte:
		return v == nil
	case map[string]any:
		if v == nil {
			return true
		}
		for _, mv := range v {
			if ValueNeedsNormalization(mv) {
				return true
			}
		}
		return false
	case []any:
		for _, ev := range v {
			if ValueNeedsNormalization(ev) {
				return true
			}
		}
		return false
	default:
		return true
	}
}

// NormalizeValue converts arbitrary Go values (including structs, pointers to
// structs, typed slices/arrays, and typed maps) into standard Go maps, slices,
// and scalars that CEL can evaluate directly. Struct tags ("json" preferred,
// falling back to "yaml", then exported field name) are respected.
func NormalizeValue(value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	if !ValueNeedsNormalization(value) {
		return value, nil
	}
	return normalizeReflectValue(reflect.ValueOf(value), nil)
}

// NormalizeReflectValue is like NormalizeValue for an already-reflected value,
// avoiding an extra interface-boxing allocation when iterating typed slices.
func NormalizeReflectValue(rv reflect.Value) (any, error) {
	return normalizeReflectValue(rv, nil)
}

func normalizeReflectValue(rv reflect.Value, seen map[visitKey]struct{}) (any, error) {
	if !rv.IsValid() {
		return nil, nil
	}

	for rv.Kind() == reflect.Interface || rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, nil
		}
		if rv.Kind() == reflect.Pointer {
			if rv.CanInterface() && implementsCustomMarshaler(rv.Type()) {
				return marshalViaJSON(rv.Interface())
			}
			ptr := rv.Pointer()
			if ptr != 0 {
				vk := visitKey{ptr: ptr, typ: rv.Type()}
				if _, exists := seen[vk]; exists {
					return nil, fmt.Errorf("cannot normalize cyclic pointer of type %s", rv.Type())
				}
				if seen == nil {
					seen = make(map[visitKey]struct{})
				}
				seen[vk] = struct{}{}
				defer delete(seen, vk)
			}
		}
		rv = rv.Elem()
	}

	if rv.CanInterface() && implementsCustomMarshaler(rv.Type()) {
		return marshalViaJSON(rv.Interface())
	}
	if rv.CanAddr() && rv.CanInterface() && implementsCustomMarshaler(reflect.PointerTo(rv.Type())) {
		return marshalViaJSON(rv.Addr().Interface())
	}

	return normalizeUnwrappedValue(rv, seen)
}

func normalizeUnwrappedValue(rv reflect.Value, seen map[visitKey]struct{}) (any, error) {
	rt := rv.Type()
	isPredeclared := rt.PkgPath() == ""

	switch rv.Kind() {
	case reflect.Bool:
		if isPredeclared && rv.CanInterface() {
			return rv.Interface(), nil
		}
		return rv.Bool(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if isPredeclared && rv.CanInterface() {
			return rv.Interface(), nil
		}
		return rv.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if rv.Kind() != reflect.Uintptr && isPredeclared && rv.CanInterface() {
			return rv.Interface(), nil
		}
		return rv.Uint(), nil
	case reflect.Float32, reflect.Float64:
		if isPredeclared && rv.CanInterface() {
			return rv.Interface(), nil
		}
		return rv.Float(), nil
	case reflect.String:
		if isPredeclared && rv.CanInterface() {
			return rv.Interface(), nil
		}
		return rv.String(), nil
	case reflect.Slice, reflect.Array:
		return normalizeSequence(rv, seen)
	case reflect.Map:
		return normalizeMap(rv, seen)
	case reflect.Struct:
		return normalizeStruct(rv, seen)
	case reflect.Interface, reflect.Pointer:
		return normalizeReflectValue(rv, seen)
	case reflect.Chan, reflect.Func, reflect.Invalid, reflect.Complex64, reflect.Complex128, reflect.UnsafePointer:
		return nil, fmt.Errorf("cannot normalize unsupported type %s", rt)
	default:
		return nil, fmt.Errorf("unsupported type: %v", rv.Kind())
	}
}

func normalizeSequence(rv reflect.Value, seen map[visitKey]struct{}) (any, error) {
	rt := rv.Type()
	if rv.Kind() == reflect.Slice && rt.Elem().Kind() == reflect.Uint8 {
		if rv.IsNil() {
			return nil, nil
		}
		cp := make([]byte, rv.Len())
		reflect.Copy(reflect.ValueOf(cp), rv)
		return cp, nil
	}

	length := rv.Len()
	normalized := make([]any, length)
	for i := range length {
		val, err := normalizeReflectValue(rv.Index(i), seen)
		if err != nil {
			return nil, fmt.Errorf("element [%d]: %w", i, err)
		}
		normalized[i] = val
	}
	return normalized, nil
}

func normalizeMap(rv reflect.Value, seen map[visitKey]struct{}) (any, error) {
	if rv.IsNil() {
		return nil, nil
	}
	if rv.CanInterface() {
		if m, ok := rv.Interface().(map[string]any); ok && !ValueNeedsNormalization(m) {
			return m, nil
		}
	}

	out := make(map[string]any, rv.Len())
	iter := rv.MapRange()
	for iter.Next() {
		k := iter.Key()
		var keyStr string
		if k.Kind() == reflect.String {
			keyStr = k.String()
		} else {
			normKey, err := normalizeReflectValue(k, seen)
			if err != nil {
				return nil, fmt.Errorf("map key: %w", err)
			}
			keyStr = fmt.Sprint(normKey)
		}
		val, err := normalizeReflectValue(iter.Value(), seen)
		if err != nil {
			return nil, fmt.Errorf("map key %q: %w", keyStr, err)
		}
		out[keyStr] = val
	}
	return out, nil
}

func normalizeStruct(rv reflect.Value, seen map[visitKey]struct{}) (map[string]any, error) {
	fields := getStructFields(rv.Type())
	out := make(map[string]any, len(fields))
	var shadowed map[string]struct{}

	hasInline := false
	for _, f := range fields {
		if f.inline != inlineNone {
			hasInline = true
			continue
		}
		if shadowed == nil {
			shadowed = make(map[string]struct{}, len(fields))
		}
		shadowed[f.key] = struct{}{}

		fv := rv.Field(f.index)
		if shouldOmitField(fv, f) {
			continue
		}
		val, err := normalizeReflectValue(fv, seen)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", f.key, err)
		}
		if f.asString {
			if formatted, ok := formatScalarAsJSONString(val); ok {
				val = formatted
			}
		}
		out[f.key] = val
	}

	if !hasInline {
		return out, nil
	}

	if shadowed == nil {
		shadowed = make(map[string]struct{}, len(fields))
	}
	for _, f := range fields {
		if f.inline == inlineNone {
			continue
		}
		fv := rv.Field(f.index)
		val, err := normalizeReflectValue(fv, seen)
		if err != nil {
			return nil, fmt.Errorf("embedded field #%d: %w", f.index, err)
		}
		inlinedMap, ok := val.(map[string]any)
		if !ok {
			continue
		}
		for k, v := range inlinedMap {
			if _, exists := shadowed[k]; !exists {
				out[k] = v
				shadowed[k] = struct{}{}
			}
		}
	}

	return out, nil
}

func getStructFields(rt reflect.Type) []structFieldInfo {
	if cached, ok := structFieldsCache.Load(rt); ok {
		if fields, ok := cached.([]structFieldInfo); ok {
			return fields
		}
	}

	n := rt.NumField()
	fields := make([]structFieldInfo, 0, n)
	for i := range n {
		sf := rt.Field(i)
		ft := sf.Type
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		// Match encoding/json: skip unexported fields EXCEPT anonymous embedded
		// struct (or pointer-to-struct) fields whose exported subfields promote.
		if !sf.IsExported() && (!sf.Anonymous || ft.Kind() != reflect.Struct) {
			continue
		}
		info, skip := parseStructField(i, sf)
		if skip {
			continue
		}
		fields = append(fields, info)
	}

	structFieldsCache.Store(rt, fields)
	return fields
}

func parseStructField(index int, sf reflect.StructField) (structFieldInfo, bool) {
	info := structFieldInfo{index: index}

	jsonTag, hasJSON := sf.Tag.Lookup("json")
	yamlTag, hasYAML := sf.Tag.Lookup("yaml")

	var explicitName bool
	var inlineOpt bool

	switch {
	case hasJSON && jsonTag != "":
		if jsonTag == "-" {
			return info, true
		}
		parts := strings.Split(jsonTag, ",")
		opts := parts[1:]
		if parts[0] != "" {
			info.key = parts[0]
			explicitName = true
		}
		info.omitEmpty = slices.Contains(opts, "omitempty")
		info.omitZero = slices.Contains(opts, "omitzero")
		info.asString = slices.Contains(opts, "string")
		inlineOpt = slices.Contains(opts, "inline")
	case hasYAML && yamlTag != "":
		if yamlTag == "-" {
			return info, true
		}
		parts := strings.Split(yamlTag, ",")
		opts := parts[1:]
		if parts[0] != "" {
			info.key = parts[0]
			explicitName = true
		}
		info.omitEmpty = slices.Contains(opts, "omitempty")
		info.omitZero = slices.Contains(opts, "omitzero")
		info.fromYAML = true
		inlineOpt = slices.Contains(opts, "inline")
	}

	if !explicitName {
		ft := sf.Type
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		if (sf.Anonymous || inlineOpt) && ft.Kind() == reflect.Struct &&
			!implementsCustomMarshaler(sf.Type) && !implementsCustomMarshaler(ft) {
			info.inline = inlineStruct
			return info, false
		}
		if inlineOpt && sf.IsExported() && ft.Kind() == reflect.Map {
			info.inline = inlineMap
			return info, false
		}
		info.key = sf.Name
	}

	// Unexported anonymous struct fields with an explicit name tag (e.g. `json:"sub"`)
	// are ignored by encoding/json rather than serialized under that key.
	if !sf.IsExported() && info.inline != inlineStruct {
		return info, true
	}

	return info, false
}

func shouldOmitField(fv reflect.Value, f structFieldInfo) bool {
	if f.omitZero {
		if z, ok := isCustomZero(fv); ok {
			return z
		}
		if fv.IsZero() {
			return true
		}
	}
	if !f.omitEmpty {
		return false
	}
	if f.fromYAML {
		if fv.Kind() == reflect.Slice || fv.Kind() == reflect.Map {
			return fv.Len() == 0
		}
		if z, ok := isCustomZero(fv); ok {
			return z
		}
		return fv.IsZero()
	}
	return isJSONEmptyValue(fv)
}

func isCustomZero(fv reflect.Value) (bool, bool) {
	if (fv.Kind() == reflect.Pointer || fv.Kind() == reflect.Interface) && fv.IsNil() {
		return true, true
	}
	if fv.CanInterface() && fv.Type().Implements(isZeroerType) {
		if z, ok := fv.Interface().(interface{ IsZero() bool }); ok {
			return z.IsZero(), true
		}
	}
	if fv.CanAddr() && fv.CanInterface() && reflect.PointerTo(fv.Type()).Implements(isZeroerType) {
		if z, ok := fv.Addr().Interface().(interface{ IsZero() bool }); ok {
			return z.IsZero(), true
		}
	}
	return false, false
}

func isJSONEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Pointer:
		return v.IsNil()
	case reflect.Struct, reflect.Chan, reflect.Func, reflect.Invalid,
		reflect.Complex64, reflect.Complex128, reflect.UnsafePointer:
		return false
	default:
		return false
	}
}

func formatScalarAsJSONString(val any) (string, bool) {
	switch v := val.(type) {
	case bool:
		return strconv.FormatBool(v), true
	case int:
		return strconv.Itoa(v), true
	case int8:
		return strconv.FormatInt(int64(v), 10), true
	case int16:
		return strconv.FormatInt(int64(v), 10), true
	case int32:
		return strconv.FormatInt(int64(v), 10), true
	case int64:
		return strconv.FormatInt(v, 10), true
	case uint:
		return strconv.FormatUint(uint64(v), 10), true
	case uint8:
		return strconv.FormatUint(uint64(v), 10), true
	case uint16:
		return strconv.FormatUint(uint64(v), 10), true
	case uint32:
		return strconv.FormatUint(uint64(v), 10), true
	case uint64:
		return strconv.FormatUint(v, 10), true
	case float32:
		return strconv.FormatFloat(float64(v), 'g', -1, 32), true
	case float64:
		return strconv.FormatFloat(v, 'g', -1, 64), true
	case string:
		b, err := json.Marshal(v)
		if err != nil {
			return "", false
		}
		return string(b), true
	default:
		return "", false
	}
}

func implementsCustomMarshaler(rt reflect.Type) bool {
	return rt.Implements(jsonMarshalerType) || rt.Implements(textMarshalerType)
}

func marshalViaJSON(v any) (any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal custom type to JSON: %w", err)
	}
	var result any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("cannot unmarshal to standard type: %w", err)
	}
	return result, nil
}
