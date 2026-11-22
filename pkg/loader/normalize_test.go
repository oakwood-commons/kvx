package loader

import (
	"encoding/json"
	"math"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type customZeroFlag struct {
	unset bool
}

func (c customZeroFlag) IsZero() bool {
	return c.unset
}

type ptrZeroFlag struct {
	unset bool
}

func (p *ptrZeroFlag) IsZero() bool {
	if p == nil {
		return true
	}
	return p.unset
}

type nonZeroCustomString string

func (nonZeroCustomString) IsZero() bool {
	return false
}

type namedOctet uint8

type ptrJSONMarshaler struct {
	Label string
}

func (p *ptrJSONMarshaler) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{"custom": p.Label})
}

type unexportedEmbeddedBase struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	secret  string
}

func TestNormalizeValue_TagsAndEmbedding(t *testing.T) {
	type EmbeddedBase struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	type YAMLOnly struct {
		ServiceName string            `yaml:"service_name"`
		Port        int               `yaml:"port,omitempty"`
		EmptyTags   []string          `yaml:"empty_tags,omitempty"`
		EmptyLabels map[string]string `yaml:"empty_labels,omitempty"`
		ZeroFlag    customZeroFlag    `yaml:"zero_flag,omitempty"`
		Ignored     string            `yaml:"-"`
		Extra       map[string]string `yaml:",inline"`
	}

	type MixedChild struct {
		EmbeddedBase
		Name        string         `json:"name"`
		Code        int            `json:"code,string"`
		Flag        bool           `json:"flag,string"`
		Rate        float64        `json:"rate,string"`
		Quoted      string         `json:"quoted,string"`
		Optional    string         `json:"optional,omitempty"`
		ZeroOnly    int            `json:"zero_only,omitzero"`
		CustomZero  customZeroFlag `json:"custom_zero,omitzero"`
		NonUTCZero  time.Time      `json:"non_utc_zero,omitzero"`
		NilBytes    []byte         `json:"nil_bytes"`
		NilTypedMap map[string]int `json:"nil_typed_map"`
		JSONInline  map[string]any `json:",inline"`
		Secret      string         `json:"-"`
		YAMLTag     string         `yaml:"yaml_tag"`
		Raw         []byte         `json:"raw"`
		unexp       func()         // unexported unsupported field must be ignored
		YAMLSub     YAMLOnly       `json:"yaml_sub"`
	}

	t.Run("json and yaml tags plus embedding shadowing and omit rules", func(t *testing.T) {
		nonUTCZeroTime := time.Time{}.In(time.FixedZone("EST", -5*3600))
		in := MixedChild{
			EmbeddedBase: EmbeddedBase{ID: "base-1", Name: "shadowed"},
			Name:         "http",
			Code:         200,
			Flag:         true,
			Rate:         1.5,
			Quoted:       "hi",
			CustomZero:   customZeroFlag{unset: true},
			NonUTCZero:   nonUTCZeroTime,
			JSONInline:   map[string]any{"inlined_json": "ok", "name": "shadowed-json-inline"},
			Secret:       "top-secret",
			YAMLTag:      "from-yaml",
			Raw:          []byte("abc"),
			unexp:        func() {},
			YAMLSub: YAMLOnly{
				ServiceName: "gateway",
				EmptyTags:   []string{},
				EmptyLabels: map[string]string{},
				ZeroFlag:    customZeroFlag{unset: true},
				Ignored:     "skip-me",
				Extra:       map[string]string{"region": "us-east", "service_name": "shadowed-inline"},
			},
		}

		got, err := NormalizeValue(in)
		require.NoError(t, err)

		m, ok := got.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "base-1", m["id"])
		assert.Equal(t, "http", m["name"])
		assert.Equal(t, "200", m["code"])
		assert.Equal(t, "true", m["flag"])
		assert.Equal(t, "1.5", m["rate"])
		assert.Equal(t, `"hi"`, m["quoted"])
		assert.Equal(t, "ok", m["inlined_json"])
		assert.Equal(t, "from-yaml", m["yaml_tag"])
		assert.Equal(t, []byte("abc"), m["raw"])
		assert.True(t, m["nil_bytes"] == nil, "nil []byte should normalize to untyped nil")
		assert.True(t, m["nil_typed_map"] == nil, "nil map should normalize to untyped nil")
		assert.NotContains(t, m, "optional")
		assert.NotContains(t, m, "zero_only")
		assert.NotContains(t, m, "custom_zero")
		assert.NotContains(t, m, "non_utc_zero")
		assert.NotContains(t, m, "Secret")

		sub, ok := m["yaml_sub"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "gateway", sub["service_name"])
		assert.Equal(t, "us-east", sub["region"])
		assert.NotContains(t, sub, "port")
		assert.NotContains(t, sub, "empty_tags")
		assert.NotContains(t, sub, "empty_labels")
		assert.NotContains(t, sub, "zero_flag")
		assert.NotContains(t, sub, "Ignored")
	})

	t.Run("embedded unexported struct by value and pointer promotes exported fields", func(t *testing.T) {
		type ByValue struct {
			unexportedEmbeddedBase
			Category string `json:"category"`
		}
		type ByPointer struct {
			*unexportedEmbeddedBase
			Category string `json:"category"`
		}
		type NamedUnexported struct {
			unexportedEmbeddedBase `json:"sub"`
			Category               string `json:"category"`
		}

		valSlice := []ByValue{
			{
				unexportedEmbeddedBase: unexportedEmbeddedBase{Name: "http", Version: "1.0.0", secret: "hidden"},
				Category:               "network",
			},
		}
		gotVal, err := NormalizeReflectValue(reflect.ValueOf(valSlice).Index(0))
		require.NoError(t, err)
		assert.Equal(t, map[string]any{
			"name":     "http",
			"version":  "1.0.0",
			"category": "network",
		}, gotVal)

		ptrSlice := []ByPointer{
			{
				unexportedEmbeddedBase: &unexportedEmbeddedBase{Name: "exec", Version: "2.0.0", secret: "hidden"},
				Category:               "system",
			},
			{
				unexportedEmbeddedBase: nil,
				Category:               "empty-base",
			},
		}
		gotPtr, err := NormalizeReflectValue(reflect.ValueOf(ptrSlice))
		require.NoError(t, err)
		assert.Equal(t, []any{
			map[string]any{"name": "exec", "version": "2.0.0", "category": "system"},
			map[string]any{"category": "empty-base"},
		}, gotPtr)

		gotNamedUnexp, err := NormalizeValue(NamedUnexported{
			unexportedEmbeddedBase: unexportedEmbeddedBase{Name: "nested", Version: "3.0.0", secret: "hidden"},
			Category:               "only-exported",
		})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{
			"sub": map[string]any{
				"name":    "nested",
				"version": "3.0.0",
			},
			"category": "only-exported",
		}, gotNamedUnexp)
	})

	t.Run("custom json marshaler like time.Time", func(t *testing.T) {
		type Event struct {
			Name string    `json:"name"`
			At   time.Time `json:"at"`
		}
		ts := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)
		got, err := NormalizeValue(Event{Name: "deploy", At: ts})
		require.NoError(t, err)

		m, ok := got.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "deploy", m["name"])
		assert.Equal(t, "2026-11-22T12:00:00Z", m["at"])
	})

	t.Run("json.RawMessage inside struct", func(t *testing.T) {
		type Envelope struct {
			Payload json.RawMessage `json:"payload"`
		}
		got, err := NormalizeValue(Envelope{Payload: json.RawMessage(`{"ok":true}`)})
		require.NoError(t, err)

		m, ok := got.(map[string]any)
		require.True(t, ok)
		payload, ok := m["payload"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, true, payload["ok"])
	})

	t.Run("named scalar types unwrap to primitives", func(t *testing.T) {
		type Status string
		type Port int
		type Item struct {
			Status Status `json:"status"`
			Port   Port   `json:"port"`
		}
		got, err := NormalizeValue(Item{Status: "ready", Port: 8080})
		require.NoError(t, err)

		m, ok := got.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "ready", m["status"])
		assert.Equal(t, int64(8080), m["port"])
	})

	t.Run("shared pointer DAG succeeds while cycle errors", func(t *testing.T) {
		type Node struct {
			Name string `json:"name"`
			Next *Node  `json:"next,omitempty"`
		}
		leaf := &Node{Name: "leaf"}
		dag := []*Node{leaf, leaf}
		got, err := NormalizeValue(dag)
		require.NoError(t, err)
		arr, ok := got.([]any)
		require.True(t, ok)
		assert.Len(t, arr, 2)

		cyclic := &Node{Name: "loop"}
		cyclic.Next = cyclic
		_, err = NormalizeValue(cyclic)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cyclic pointer")
	})

	t.Run("exported unsupported func field errors", func(t *testing.T) {
		type Bad struct {
			Callback func() `json:"callback"`
		}
		_, err := NormalizeValue(Bad{Callback: func() {}})
		require.Error(t, err)
	})

	t.Run("container cycles in maps and slices error without stack overflow", func(t *testing.T) {
		cyclicMap := map[string]any{}
		cyclicMap["self"] = cyclicMap
		assert.True(t, ValueNeedsNormalization(cyclicMap))
		_, err := NormalizeValue(cyclicMap)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cyclic map")
		_, err = LoadObject(cyclicMap)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cyclic map")

		cyclicSlice := make([]any, 1)
		cyclicSlice[0] = cyclicSlice
		assert.True(t, ValueNeedsNormalization(cyclicSlice))
		_, err = NormalizeValue(cyclicSlice)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cyclic slice")

		// Shared acyclic map/slice DAGs (including deep > cycleCheckStartDepth) succeed.
		sharedLeaf := map[string]any{"ok": true}
		sharedSlice := []any{sharedLeaf, sharedLeaf}
		cur := sharedSlice
		for range 40 {
			cur = []any{cur, sharedLeaf}
		}
		assert.False(t, ValueNeedsNormalization(cur))
		gotDAG, err := NormalizeValue(cur)
		require.NoError(t, err)
		assert.NotNil(t, gotDAG)
	})

	t.Run("named uint8 slice copies safely to byte slice", func(t *testing.T) {
		type Packet struct {
			Payload []namedOctet `json:"payload"`
			Empty   []namedOctet `json:"empty"`
		}
		got, err := NormalizeValue(Packet{
			Payload: []namedOctet{0x61, 0x62, 0x63},
			Empty:   nil,
		})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{
			"payload": []byte("abc"),
			"empty":   nil,
		}, got)
	})

	t.Run("colliding normalized map keys return error", func(t *testing.T) {
		colliding := map[any]any{
			1:   "from-int",
			"1": "from-string",
		}
		_, err := NormalizeValue(colliding)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `duplicate normalized map key "1"`)
	})

	t.Run("embedded field dominance by depth tag precedence and ambiguity", func(t *testing.T) {
		type Deep struct {
			Name string `json:"name"`
		}
		type Mid struct {
			Deep
		}
		type Shallow struct {
			Name string `json:"name"`
		}
		// Mid appears before Shallow, yet Shallow is shallower (depth 2 vs 3).
		type DepthWins struct {
			Mid
			Shallow
		}
		gotDepth, err := NormalizeValue(DepthWins{
			Mid:     Mid{Deep: Deep{Name: "from-deep"}},
			Shallow: Shallow{Name: "from-shallow"},
		})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"name": "from-shallow"}, gotDepth)

		// Equal-depth conflict with both tagged is omitted as ambiguous.
		type A struct {
			Name string `json:"name"`
		}
		type B struct {
			Name string `json:"name"`
		}
		type EqualTaggedAmbiguous struct {
			A
			B           //nolint:govet // intentionally tests encoding/json duplicate embedded tag ambiguity
			Keep string `json:"keep"`
		}
		gotAmb, err := NormalizeValue(EqualTaggedAmbiguous{
			A:    A{Name: "a"},
			B:    B{Name: "b"},
			Keep: "yes",
		})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"keep": "yes"}, gotAmb)

		// Equal-depth tie broken by explicit tag over untagged field name.
		type Untagged struct {
			Title string
		}
		type Tagged struct {
			Title string `json:"Title"`
		}
		type TagTieBreak struct {
			Untagged
			Tagged
		}
		gotTag, err := NormalizeValue(TagTieBreak{
			Untagged: Untagged{Title: "untagged"},
			Tagged:   Tagged{Title: "tagged"},
		})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"Title": "tagged"}, gotTag)

		// Diamond embedding omits ambiguous promoted fields just like encoding/json.
		type DiamondLeaf struct {
			Shared string `json:"shared"`
		}
		type Left struct {
			DiamondLeaf
		}
		type Right struct {
			DiamondLeaf
		}
		type Diamond struct {
			Left
			Right        //nolint:govet // intentionally tests encoding/json diamond embedding ambiguity
			ID    string `json:"id"`
		}
		gotDiamond, err := NormalizeValue(Diamond{
			Left:  Left{DiamondLeaf: DiamondLeaf{Shared: "x"}},
			Right: Right{DiamondLeaf: DiamondLeaf{Shared: "x"}},
			ID:    "d1",
		})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"id": "d1"}, gotDiamond)
	})

	t.Run("omitzero false custom IsZero still honors omitempty and pointer receiver parity", func(t *testing.T) {
		type Record struct {
			BothEmpty    nonZeroCustomString `json:"both_empty,omitempty,omitzero"`
			ZeroOnly     nonZeroCustomString `json:"zero_only,omitzero"`
			PtrZeroTrue  ptrZeroFlag         `json:"ptr_zero_true,omitzero"`
			PtrZeroFalse ptrZeroFlag         `json:"ptr_zero_false,omitzero"`
			IfaceNilPtr  isZeroer            `json:"iface_nil_ptr,omitzero"`
		}

		var nilPtr *ptrZeroFlag
		rec := Record{
			BothEmpty:    "",
			ZeroOnly:     "",
			PtrZeroTrue:  ptrZeroFlag{unset: true},
			PtrZeroFalse: ptrZeroFlag{unset: false},
			IfaceNilPtr:  nilPtr,
		}

		// Unaddressable struct value (e.g. boxed in any)
		gotBoxed, err := NormalizeValue(rec)
		require.NoError(t, err)

		// Addressable struct element inside typed slice
		typedSlice := []Record{rec}
		gotAddr, err := NormalizeReflectValue(reflect.ValueOf(typedSlice).Index(0))
		require.NoError(t, err)

		expected := map[string]any{
			"zero_only":      "",
			"ptr_zero_false": map[string]any{},
		}
		assert.Equal(t, expected, gotBoxed)
		assert.Equal(t, expected, gotAddr)
	})

	t.Run("encodeJSONString matches encoding/json across special characters and scalars", func(t *testing.T) {
		samples := []string{
			"",
			"plain-ascii",
			`with "quotes" and \backslash\`,
			"controls \b\f\n\r\t\x00\x1f",
			"html <script>&</script>",
			"utf8 cafe \u00e9 \U0001f680",
			"js-line-separators \u2028 \u2029",
			string([]byte{0xff, 'a', 0xfe}),
		}
		for _, s := range samples {
			stdBytes, err := json.Marshal(s)
			require.NoError(t, err)
			assert.Equal(t, string(stdBytes), encodeJSONString(s), "mismatch for %q", s)
		}

		type AllStringTaggedScalars struct {
			I8  int8    `json:"i8,string"`
			I16 int16   `json:"i16,string"`
			I32 int32   `json:"i32,string"`
			I64 int64   `json:"i64,string"`
			U   uint    `json:"u,string"`
			U8  uint8   `json:"u8,string"`
			U16 uint16  `json:"u16,string"`
			U32 uint32  `json:"u32,string"`
			U64 uint64  `json:"u64,string"`
			F32 float32 `json:"f32,string"`
		}
		gotScalars, err := NormalizeValue(AllStringTaggedScalars{
			I8: -8, I16: -16, I32: -32, I64: -64,
			U: 1, U8: 8, U16: 16, U32: 32, U64: 64,
			F32: 2.5,
		})
		require.NoError(t, err)
		assert.Equal(t, map[string]any{
			"i8": "-8", "i16": "-16", "i32": "-32", "i64": "-64",
			"u": "1", "u8": "8", "u16": "16", "u32": "32", "u64": "64",
			"f32": "2.5",
		}, gotScalars)
	})

	t.Run("pointer-receiver marshaler parity, json.Number, string option eligibility, non-zero any interface, and JSON float formatting", func(t *testing.T) {
		pm := ptrJSONMarshaler{Label: "ok"}
		gotBoxedPM, err := NormalizeValue(pm)
		require.NoError(t, err)
		pmSlice := []ptrJSONMarshaler{pm}
		gotAddrPM, err := NormalizeReflectValue(reflect.ValueOf(pmSlice).Index(0))
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"custom": "ok"}, gotBoxedPM)
		assert.Equal(t, gotAddrPM, gotBoxedPM)

		ts := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)
		var typedNilIntPtr *int
		codeVal := 200

		type EdgePayload struct {
			Count      json.Number `json:"count"`
			EmptyNum   json.Number `json:"empty_num"`
			StrNum     json.Number `json:"str_num,string"`
			AnyInt     any         `json:"any_int,string"`      //nolint:staticcheck // intentionally tests encoding/json ignoring ,string on interface fields
			TimeStrTag time.Time   `json:"time_str_tag,string"` //nolint:staticcheck // intentionally tests encoding/json ignoring ,string on custom marshalers
			PtrCode    *int        `json:"ptr_code,string"`
			NonZeroAny any         `json:"non_zero_any,omitzero"`
			NilAny     any         `json:"nil_any,omitzero"`
			BigFloat64 float64     `json:"big_f64,string"`
			SmallF64   float64     `json:"small_f64,string"`
			SciF64     float64     `json:"sci_f64,string"`
			BigFloat32 float32     `json:"big_f32,string"`
		}

		gotEdge, err := NormalizeValue(EdgePayload{
			Count:      json.Number("42"),
			EmptyNum:   json.Number(""),
			StrNum:     json.Number("9007199254740993"),
			AnyInt:     42,
			TimeStrTag: ts,
			PtrCode:    &codeVal,
			NonZeroAny: typedNilIntPtr,
			NilAny:     nil,
			BigFloat64: 1000000,
			SmallF64:   1e-7,
			SciF64:     1e21,
			BigFloat32: 1000000,
		})
		require.NoError(t, err)

		m, ok := gotEdge.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, float64(42), m["count"])
		assert.Equal(t, float64(0), m["empty_num"])
		assert.Equal(t, "9007199254740993", m["str_num"])
		assert.Equal(t, 42, m["any_int"])
		assert.Equal(t, "2026-11-22T12:00:00Z", m["time_str_tag"])
		assert.Equal(t, "200", m["ptr_code"])
		valAny, hasNonZeroAny := m["non_zero_any"]
		assert.True(t, hasNonZeroAny, "non-nil interface holding typed nil pointer must be retained under omitzero")
		assert.True(t, valAny == nil, "typed nil pointer inside any should normalize its value to untyped nil")
		assert.NotContains(t, m, "nil_any")
		assert.Equal(t, "1000000", m["big_f64"])
		assert.Equal(t, "1e-7", m["small_f64"])
		assert.Equal(t, "1e+21", m["sci_f64"])
		assert.Equal(t, "1000000", m["big_f32"])

		type BadNumber struct {
			N json.Number `json:"n"`
		}
		_, err = NormalizeValue(BadNumber{N: json.Number("not-a-number")})
		require.Error(t, err)

		type NaNStringFloat struct {
			F float64 `json:"f,string"`
		}
		_, err = NormalizeValue(NaNStringFloat{F: math.NaN()})
		require.Error(t, err)
	})
}

func TestValueNeedsNormalization(t *testing.T) {
	type sample struct {
		Name string `json:"name"`
	}
	type namedStr string

	tests := []struct {
		name  string
		value any
		want  bool
	}{
		{"nil interface", nil, false},
		{"plain string", "hello", false},
		{"plain bool", true, false},
		{"plain int64", int64(42), false},
		{"plain float64", 3.14, false},
		{"non-nil []byte", []byte("abc"), false},
		{"nil []byte", []byte(nil), true},
		{"map[string]any of scalars", map[string]any{"k": "v", "n": 1}, false},
		{"nil map[string]any", map[string]any(nil), true},
		{"map[string]any containing struct", map[string]any{"s": sample{Name: "a"}}, true},
		{"[]any of scalars and maps", []any{"a", map[string]any{"k": 1}}, false},
		{"[]any containing pointer to struct", []any{&sample{Name: "a"}}, true},
		{"typed struct", sample{Name: "a"}, true},
		{"typed slice []string", []string{"a", "b"}, true},
		{"named string scalar", namedStr("custom"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ValueNeedsNormalization(tt.value))
		})
	}
}

func TestNormalizeReflectValue_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		rv       reflect.Value
		expected any
	}{
		{
			name:     "invalid zero reflect.Value",
			rv:       reflect.Value{},
			expected: nil,
		},
		{
			name:     "non-string keyed map converts keys to string",
			rv:       reflect.ValueOf(map[int]string{1: "one", 2: "two"}),
			expected: map[string]any{"1": "one", "2": "two"},
		},
		{
			name:     "typed nil map returns untyped nil",
			rv:       reflect.ValueOf(map[string]string(nil)),
			expected: nil,
		},
		{
			name:     "typed nil byte slice returns untyped nil",
			rv:       reflect.ValueOf([]byte(nil)),
			expected: nil,
		},
		{
			name:     "uintptr converts to uint64",
			rv:       reflect.ValueOf(uintptr(7)),
			expected: uint64(7),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeReflectValue(tt.rv)
			require.NoError(t, err)
			if tt.expected == nil {
				assert.True(t, got == nil, "expected untyped nil, got %T", got)
				return
			}
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestNormalizeValue_ConcurrentRace(t *testing.T) {
	type Summary struct {
		unexportedEmbeddedBase
		Category   string `json:"category" yaml:"category"`
		Deprecated bool   `json:"deprecated" yaml:"deprecated"`
	}

	slice := []Summary{
		{
			unexportedEmbeddedBase: unexportedEmbeddedBase{Name: "http", Version: "1.0.0"},
			Category:               "network",
		},
		{
			unexportedEmbeddedBase: unexportedEmbeddedBase{Name: "exec", Version: "1.2.0"},
			Category:               "system",
			Deprecated:             true,
		},
	}

	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 50 {
				got, err := NormalizeValue(slice)
				assert.NoError(t, err)
				assert.Len(t, got, 2)

				elem, rErr := NormalizeReflectValue(reflect.ValueOf(slice).Index(0))
				assert.NoError(t, rErr)
				assert.NotNil(t, elem)
			}
		}()
	}
	wg.Wait()
}

func BenchmarkNormalizeValue(b *testing.B) {
	type Summary struct {
		Name       string `json:"name" yaml:"name"`
		Category   string `json:"category" yaml:"category"`
		Deprecated bool   `json:"deprecated" yaml:"deprecated"`
	}
	item := Summary{Name: "http", Category: "network"}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := NormalizeValue(item); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNormalizeReflectValue(b *testing.B) {
	type Summary struct {
		Name       string `json:"name" yaml:"name"`
		Category   string `json:"category" yaml:"category"`
		Deprecated bool   `json:"deprecated" yaml:"deprecated"`
	}
	items := make([]Summary, 100)
	for i := range items {
		items[i] = Summary{Name: "http", Category: "network"}
	}
	rv := reflect.ValueOf(items)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for i := range rv.Len() {
			if _, err := NormalizeReflectValue(rv.Index(i)); err != nil {
				b.Fatal(err)
			}
		}
	}
}
