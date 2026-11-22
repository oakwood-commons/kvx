package loader

import (
	"cmp"
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

// cycleCheckStartDepth defers seen-map allocation in ValueNeedsNormalization
// until recursion exceeds typical acyclic nesting depths (matching the spirit
// of encoding/json's startDetectingCyclesAfter) while bounding cyclic inputs.
const cycleCheckStartDepth = 32

const hexDigits = "0123456789abcdef"

type isZeroer interface {
	IsZero() bool
}

var (
	byteType          = reflect.TypeFor[byte]()
	mapStringAnyType  = reflect.TypeFor[map[string]any]()
	sliceAnyType      = reflect.TypeFor[[]any]()
	jsonMarshalerType = reflect.TypeFor[json.Marshaler]()
	textMarshalerType = reflect.TypeFor[encoding.TextMarshaler]()
	isZeroerType      = reflect.TypeFor[isZeroer]()
	structFieldsCache sync.Map // map[reflect.Type]structFieldsPlan
)

type visitKey struct {
	ptr  uintptr
	typ  reflect.Type
	slen int
}

type inlineKind uint8

const (
	inlineNone inlineKind = iota
	inlineStruct
	inlineMap
)

type structFieldInfo struct {
	index     []int
	key       string
	tagged    bool
	omitEmpty bool
	omitZero  bool
	asString  bool
	fromYAML  bool
}

type structFieldsPlan struct {
	fields     []structFieldInfo
	inlineMaps []structFieldInfo
}

type embedQueueItem struct {
	typ   reflect.Type
	index []int
}

// ValueNeedsNormalization reports whether value (or a nested element within a
// []any or map[string]any) contains Go types such as structs, pointers, typed
// slices/maps, nil byte slices, named scalars, or reference cycles that must be
// processed by NormalizeValue before CEL evaluation.
func ValueNeedsNormalization(value any) bool {
	return valueNeedsNormalization(value, 0, nil)
}

func valueNeedsNormalization(value any, depth int, seen map[visitKey]struct{}) bool {
	switch v := value.(type) {
	case nil, string, bool,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return false
	case []byte:
		return v == nil
	case map[string]any:
		return mapNeedsNormalization(v, depth, seen)
	case []any:
		return sliceNeedsNormalization(v, depth, seen)
	default:
		return true
	}
}

func mapNeedsNormalization(v map[string]any, depth int, seen map[visitKey]struct{}) bool {
	if v == nil {
		return true
	}
	if len(v) == 0 {
		return false
	}
	if depth >= cycleCheckStartDepth {
		ptr := reflect.ValueOf(v).Pointer()
		if ptr != 0 {
			vk := visitKey{ptr: ptr, typ: mapStringAnyType}
			if _, exists := seen[vk]; exists {
				return true
			}
			if seen == nil {
				seen = make(map[visitKey]struct{})
			}
			seen[vk] = struct{}{}
			defer delete(seen, vk)
		}
	}
	for _, mv := range v {
		if valueNeedsNormalization(mv, depth+1, seen) {
			return true
		}
	}
	return false
}

func sliceNeedsNormalization(v []any, depth int, seen map[visitKey]struct{}) bool {
	if len(v) == 0 {
		return false
	}
	if depth >= cycleCheckStartDepth {
		ptr := reflect.ValueOf(v).Pointer()
		if ptr != 0 {
			vk := visitKey{ptr: ptr, typ: sliceAnyType, slen: len(v)}
			if _, exists := seen[vk]; exists {
				return true
			}
			if seen == nil {
				seen = make(map[visitKey]struct{})
			}
			seen[vk] = struct{}{}
			defer delete(seen, vk)
		}
	}
	for _, ev := range v {
		if valueNeedsNormalization(ev, depth+1, seen) {
			return true
		}
	}
	return false
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
		elemType := rt.Elem()
		if !implementsCustomMarshaler(elemType) && !implementsCustomMarshaler(reflect.PointerTo(elemType)) {
			return copyByteLikeSlice(rv, elemType), nil
		}
	}

	length := rv.Len()
	if rv.Kind() == reflect.Slice && length > 0 {
		ptr := rv.Pointer()
		if ptr != 0 {
			vk := visitKey{ptr: ptr, typ: rt, slen: length}
			if _, exists := seen[vk]; exists {
				return nil, fmt.Errorf("cannot normalize cyclic slice of type %s", rt)
			}
			if seen == nil {
				seen = make(map[visitKey]struct{})
			}
			seen[vk] = struct{}{}
			defer delete(seen, vk)
		}
	}

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

func copyByteLikeSlice(rv reflect.Value, elemType reflect.Type) any {
	if rv.IsNil() {
		return nil
	}
	cp := make([]byte, rv.Len())
	if elemType == byteType {
		reflect.Copy(reflect.ValueOf(cp), rv)
		return cp
	}
	for i := range rv.Len() {
		cp[i] = byte(rv.Index(i).Uint() & 0xff)
	}
	return cp
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

	ptr := rv.Pointer()
	if ptr != 0 {
		vk := visitKey{ptr: ptr, typ: rv.Type()}
		if _, exists := seen[vk]; exists {
			return nil, fmt.Errorf("cannot normalize cyclic map of type %s", rv.Type())
		}
		if seen == nil {
			seen = make(map[visitKey]struct{})
		}
		seen[vk] = struct{}{}
		defer delete(seen, vk)
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
		if _, exists := out[keyStr]; exists {
			return nil, fmt.Errorf("duplicate normalized map key %q", keyStr)
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
	plan := getStructFieldsPlan(rv.Type())
	out := make(map[string]any, len(plan.fields))

	var shadowed map[string]struct{}
	if len(plan.inlineMaps) > 0 {
		shadowed = make(map[string]struct{}, len(plan.fields))
	}

	for _, f := range plan.fields {
		fv, ok := fieldByIndex(rv, f.index)
		if !ok {
			continue
		}
		if shadowed != nil {
			shadowed[f.key] = struct{}{}
		}
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

	for _, f := range plan.inlineMaps {
		fv, ok := fieldByIndex(rv, f.index)
		if !ok {
			continue
		}
		val, err := normalizeReflectValue(fv, seen)
		if err != nil {
			return nil, fmt.Errorf("embedded field #%v: %w", f.index, err)
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

func fieldByIndex(rv reflect.Value, index []int) (reflect.Value, bool) {
	fv := rv
	for _, idx := range index {
		if fv.Kind() == reflect.Pointer {
			if fv.IsNil() {
				return reflect.Value{}, false
			}
			fv = fv.Elem()
		}
		fv = fv.Field(idx)
	}
	return fv, true
}

func getStructFieldsPlan(rt reflect.Type) structFieldsPlan {
	if cached, ok := structFieldsCache.Load(rt); ok {
		if plan, ok := cached.(structFieldsPlan); ok {
			return plan
		}
	}

	plan := buildStructFieldsPlan(rt)
	structFieldsCache.Store(rt, plan)
	return plan
}

func buildStructFieldsPlan(rt reflect.Type) structFieldsPlan {
	current := []embedQueueItem{}
	next := []embedQueueItem{{typ: rt}}
	var count, nextCount map[reflect.Type]int
	visited := make(map[reflect.Type]bool)

	var candidates []structFieldInfo
	var inlineMaps []structFieldInfo

	for len(next) > 0 {
		current, next = next, current[:0]
		count, nextCount = nextCount, make(map[reflect.Type]int)

		for _, item := range current {
			if visited[item.typ] {
				continue
			}
			visited[item.typ] = true
			scanStructTypeFields(item, count[item.typ] > 1, nextCount, &next, &candidates, &inlineMaps)
		}
	}

	return structFieldsPlan{
		fields:     resolveDominantFields(candidates),
		inlineMaps: inlineMaps,
	}
}

func scanStructTypeFields(
	item embedQueueItem,
	duplicated bool,
	nextCount map[reflect.Type]int,
	next *[]embedQueueItem,
	candidates *[]structFieldInfo,
	inlineMaps *[]structFieldInfo,
) {
	for i := range item.typ.NumField() {
		sf := item.typ.Field(i)
		ft := sf.Type
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		// Match encoding/json: skip unexported fields EXCEPT anonymous embedded
		// struct (or pointer-to-struct) fields, which either promote exported
		// subfields or serialize exported subfields under an explicit tag.
		if sf.Anonymous {
			if !sf.IsExported() && ft.Kind() != reflect.Struct {
				continue
			}
		} else if !sf.IsExported() {
			continue
		}

		idx := make([]int, len(item.index)+1)
		copy(idx, item.index)
		idx[len(item.index)] = i

		info, inline, skip := parseStructField(idx, sf, ft)
		if skip {
			continue
		}

		switch inline {
		case inlineStruct:
			nextCount[ft]++
			if nextCount[ft] == 1 {
				*next = append(*next, embedQueueItem{typ: ft, index: idx})
			}
		case inlineMap:
			if !duplicated {
				*inlineMaps = append(*inlineMaps, info)
			}
		case inlineNone:
			*candidates = append(*candidates, info)
			if duplicated {
				*candidates = append(*candidates, info)
			}
		}
	}
}

func resolveDominantFields(candidates []structFieldInfo) []structFieldInfo {
	if len(candidates) == 0 {
		return nil
	}

	slices.SortFunc(candidates, func(a, b structFieldInfo) int {
		if c := strings.Compare(a.key, b.key); c != 0 {
			return c
		}
		if c := cmp.Compare(len(a.index), len(b.index)); c != 0 {
			return c
		}
		if a.tagged != b.tagged {
			if a.tagged {
				return -1
			}
			return 1
		}
		return slices.Compare(a.index, b.index)
	})

	out := make([]structFieldInfo, 0, len(candidates))
	for advance, i := 0, 0; i < len(candidates); i += advance {
		key := candidates[i].key
		for advance = 1; i+advance < len(candidates); advance++ {
			if candidates[i+advance].key != key {
				break
			}
		}
		group := candidates[i : i+advance]
		if len(group) == 1 {
			out = append(out, group[0])
			continue
		}
		if dominant, ok := dominantStructField(group); ok {
			out = append(out, dominant)
		}
	}

	slices.SortFunc(out, func(a, b structFieldInfo) int {
		return slices.Compare(a.index, b.index)
	})
	return out
}

func dominantStructField(group []structFieldInfo) (structFieldInfo, bool) {
	if len(group) > 1 && len(group[0].index) == len(group[1].index) && group[0].tagged == group[1].tagged {
		return structFieldInfo{}, false
	}
	return group[0], true
}

func parseStructField(index []int, sf reflect.StructField, ft reflect.Type) (structFieldInfo, inlineKind, bool) {
	info := structFieldInfo{index: index}

	jsonTag, hasJSON := sf.Tag.Lookup("json")
	yamlTag, hasYAML := sf.Tag.Lookup("yaml")

	var inlineOpt bool

	switch {
	case hasJSON && jsonTag != "":
		if jsonTag == "-" {
			return info, inlineNone, true
		}
		parts := strings.Split(jsonTag, ",")
		opts := parts[1:]
		if parts[0] != "" {
			info.key = parts[0]
			info.tagged = true
		}
		info.omitEmpty = slices.Contains(opts, "omitempty")
		info.omitZero = slices.Contains(opts, "omitzero")
		info.asString = slices.Contains(opts, "string")
		inlineOpt = slices.Contains(opts, "inline")
	case hasYAML && yamlTag != "":
		if yamlTag == "-" {
			return info, inlineNone, true
		}
		parts := strings.Split(yamlTag, ",")
		opts := parts[1:]
		if parts[0] != "" {
			info.key = parts[0]
			info.tagged = true
		}
		info.omitEmpty = slices.Contains(opts, "omitempty")
		info.omitZero = slices.Contains(opts, "omitzero")
		info.fromYAML = true
		inlineOpt = slices.Contains(opts, "inline")
	}

	if !info.tagged {
		if (sf.Anonymous || inlineOpt) && ft.Kind() == reflect.Struct &&
			!implementsCustomMarshaler(sf.Type) && !implementsCustomMarshaler(ft) {
			return info, inlineStruct, false
		}
		if inlineOpt && sf.IsExported() && ft.Kind() == reflect.Map {
			return info, inlineMap, false
		}
		info.key = sf.Name
	}

	return info, inlineNone, false
}

func shouldOmitField(fv reflect.Value, f structFieldInfo) bool {
	if f.omitZero {
		if z, ok := isCustomZero(fv); ok {
			if z {
				return true
			}
		} else if fv.IsZero() {
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
	if fv.Kind() == reflect.Interface {
		if fv.IsNil() || (fv.Elem().Kind() == reflect.Pointer && fv.Elem().IsNil()) {
			return true, true
		}
	}
	if fv.Kind() == reflect.Pointer && fv.IsNil() {
		return true, true
	}
	if !fv.CanInterface() {
		return false, false
	}
	ft := fv.Type()
	if ft.Implements(isZeroerType) {
		if z, ok := fv.Interface().(isZeroer); ok {
			return z.IsZero(), true
		}
	}
	if reflect.PointerTo(ft).Implements(isZeroerType) {
		addrTarget := fv
		if !addrTarget.CanAddr() {
			boxed := reflect.New(ft).Elem()
			boxed.Set(fv)
			addrTarget = boxed
		}
		if z, ok := addrTarget.Addr().Interface().(isZeroer); ok {
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
		return encodeJSONString(v), true
	default:
		return "", false
	}
}

// encodeJSONString formats s as an RFC 8259 JSON string literal matching
// encoding/json.Marshal (including HTML and U+2028/U+2029 escaping) without
// invoking json.Marshal.
func encodeJSONString(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('"')
	for i := 0; i < len(s); {
		c := s[i]
		if c < utf8.RuneSelf {
			writeASCIIJSONByte(&b, c)
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			b.WriteRune(utf8.RuneError)
			i++
			continue
		}
		if r == '\u2028' || r == '\u2029' {
			b.WriteString(`\u202`)
			b.WriteByte(hexDigits[r&0x0f])
			i += size
			continue
		}
		b.WriteString(s[i : i+size])
		i += size
	}
	b.WriteByte('"')
	return b.String()
}

func writeASCIIJSONByte(b *strings.Builder, c byte) {
	switch c {
	case '\\', '"':
		b.WriteByte('\\')
		b.WriteByte(c)
	case '\b':
		b.WriteString(`\b`)
	case '\f':
		b.WriteString(`\f`)
	case '\n':
		b.WriteString(`\n`)
	case '\r':
		b.WriteString(`\r`)
	case '\t':
		b.WriteString(`\t`)
	case '<', '>', '&':
		b.WriteString(`\u00`)
		b.WriteByte(hexDigits[c>>4])
		b.WriteByte(hexDigits[c&0x0f])
	default:
		if c < 0x20 {
			b.WriteString(`\u00`)
			b.WriteByte(hexDigits[c>>4])
			b.WriteByte(hexDigits[c&0x0f])
		} else {
			b.WriteByte(c)
		}
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
