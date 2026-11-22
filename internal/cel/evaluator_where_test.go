package cel

import (
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluateWhere(t *testing.T) {
	eval, err := NewEvaluator()
	require.NoError(t, err)

	tests := []struct {
		name     string
		expr     string
		data     interface{}
		expected []interface{}
		wantErr  string
	}{
		{
			name: "filter maps by field",
			expr: `_.type == "oci"`,
			data: []interface{}{
				map[string]interface{}{"name": "a", "type": "oci"},
				map[string]interface{}{"name": "b", "type": "helm"},
				map[string]interface{}{"name": "c", "type": "oci"},
			},
			expected: []interface{}{
				map[string]interface{}{"name": "a", "type": "oci"},
				map[string]interface{}{"name": "c", "type": "oci"},
			},
		},
		{
			name:     "filter scalars",
			expr:     `_ > 3`,
			data:     []interface{}{int64(1), int64(2), int64(5), int64(4)},
			expected: []interface{}{int64(5), int64(4)},
		},
		{
			name: "no matches returns empty",
			expr: `_.active == true`,
			data: []interface{}{
				map[string]interface{}{"active": false},
			},
			expected: []interface{}{},
		},
		{
			name: "all match",
			expr: `_.ok == true`,
			data: []interface{}{
				map[string]interface{}{"ok": true},
				map[string]interface{}{"ok": true},
			},
			expected: []interface{}{
				map[string]interface{}{"ok": true},
				map[string]interface{}{"ok": true},
			},
		},
		{
			name:     "empty list",
			expr:     `_ > 0`,
			data:     []interface{}{},
			expected: []interface{}{},
		},
		{
			name: "nested field access",
			expr: `_.metadata.env == "prod"`,
			data: []interface{}{
				map[string]interface{}{"metadata": map[string]interface{}{"env": "prod"}},
				map[string]interface{}{"metadata": map[string]interface{}{"env": "dev"}},
			},
			expected: []interface{}{
				map[string]interface{}{"metadata": map[string]interface{}{"env": "prod"}},
			},
		},
		{
			name:    "non-list input errors",
			expr:    `_.name == "x"`,
			data:    map[string]interface{}{"name": "x"},
			wantErr: "EvaluateWhere requires list data",
		},
		{
			name:    "non-boolean expression errors at runtime",
			expr:    `_.name`,
			data:    []interface{}{map[string]interface{}{"name": "x"}},
			wantErr: "where filter expression must return a boolean",
		},
		{
			name:    "non-boolean expression rejected at compile time",
			expr:    `size(_)`,
			data:    []interface{}{map[string]interface{}{"name": "x"}},
			wantErr: "where filter expression must return a boolean, got int",
		},
		{
			name:    "non-boolean expression on empty list rejected",
			expr:    `_.name + "x"`,
			data:    []interface{}{},
			wantErr: "where filter expression must return a boolean, got string",
		},
		{
			name:    "invalid CEL syntax errors",
			expr:    `_.name ==`,
			data:    []interface{}{map[string]interface{}{"name": "x"}},
			wantErr: "where filter compilation error",
		},
		{
			name: "typed map slice input",
			expr: `_.count > 1`,
			data: []map[string]interface{}{
				{"count": int64(2)},
				{"count": int64(0)},
			},
			expected: []interface{}{
				map[string]interface{}{"count": int64(2)},
			},
		},
		{
			name: "missing key treated as nil not error",
			expr: `_.group == "core"`,
			data: []interface{}{
				map[string]interface{}{"name": "foo", "group": "core"},
				map[string]interface{}{"name": "bar"},
			},
			expected: []interface{}{
				map[string]interface{}{"name": "foo", "group": "core"},
			},
		},
		{
			name: "all items missing filtered key returns empty",
			expr: `_.group == "core"`,
			data: []interface{}{
				map[string]interface{}{"name": "foo"},
				map[string]interface{}{"name": "bar"},
			},
			expected: []interface{}{},
		},
		{
			name: "compound expression matches via other branch despite missing key",
			expr: `_.group == "core" || _.name == "bar"`,
			data: []interface{}{
				map[string]interface{}{"name": "foo", "group": "core"},
				map[string]interface{}{"name": "bar"},
			},
			expected: []interface{}{
				map[string]interface{}{"name": "foo", "group": "core"},
				map[string]interface{}{"name": "bar"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := eval.EvaluateWhere(tt.expr, tt.data)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvaluateWhere_NoMutation(t *testing.T) {
	eval, err := NewEvaluator()
	require.NoError(t, err)

	original := map[string]interface{}{"name": "bar"}
	data := []interface{}{
		map[string]interface{}{"name": "foo", "group": "core"},
		original,
	}

	_, err = eval.EvaluateWhere(`_.group == "core"`, data)
	require.NoError(t, err)

	// The original map must not have been modified.
	assert.Equal(t, map[string]interface{}{"name": "bar"}, original)
	_, hasGroup := original["group"]
	assert.False(t, hasGroup, "original map should not have gained a 'group' key")
}

func BenchmarkEvaluateWhere(b *testing.B) {
	eval, err := NewEvaluator()
	require.NoError(b, err)

	items := make([]interface{}, 1000)
	for i := range items {
		items[i] = map[string]interface{}{
			"index":  int64(i),
			"active": i%2 == 0,
		}
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, err := eval.EvaluateWhere(`_.active == true`, items)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestEvaluateWhere_TypedSlicesAndNonListSentinel(t *testing.T) {
	type Summary struct {
		Name       string `json:"name" yaml:"name"`
		Category   string `json:"category" yaml:"category"`
		Deprecated bool   `json:"deprecated" yaml:"deprecated"`
	}
	type unexportedBase struct {
		Name string `json:"name"`
	}
	type EmbeddedUnexportedSummary struct {
		unexportedBase
		Category string `json:"category"`
	}
	type UnnormalizableItem struct {
		Callback func() `json:"callback"`
	}
	type YAMLOnlySummary struct {
		ServiceName string `yaml:"service_name"`
		Enabled     bool   `yaml:"enabled"`
	}
	type rawBytes []byte

	var nilSummarySlice []Summary
	var nilPtrSlice []*Summary
	var nilAnySlice []any
	var nilByteSlice []byte

	eval, err := NewEvaluator()
	require.NoError(t, err)

	tests := []struct {
		name          string
		expr          string
		data          any
		expected      []any
		wantNotList   bool
		wantErrSubstr string
	}{
		{
			name: "typed struct slice []Summary",
			expr: `_.name == "http"`,
			data: []Summary{
				{Name: "http", Category: "network"},
				{Name: "exec", Category: "system"},
			},
			expected: []any{
				map[string]any{"name": "http", "category": "network", "deprecated": false},
			},
		},
		{
			name: "pointer-to-struct slice []*Summary with nil element",
			expr: `_.name == "http"`,
			data: []*Summary{
				nil,
				{Name: "http", Category: "network"},
				{Name: "exec", Category: "system"},
			},
			expected: []any{
				map[string]any{"name": "http", "category": "network", "deprecated": false},
			},
		},
		{
			name: "yaml-only struct tags",
			expr: `_.service_name == "auth" && _.enabled`,
			data: []YAMLOnlySummary{
				{ServiceName: "auth", Enabled: true},
				{ServiceName: "billing", Enabled: false},
			},
			expected: []any{
				map[string]any{"service_name": "auth", "enabled": true},
			},
		},
		{
			name: "fixed-size struct array",
			expr: `_.name == "http"`,
			data: [2]Summary{
				{Name: "http", Category: "network"},
				{Name: "exec", Category: "system"},
			},
			expected: []any{
				map[string]any{"name": "http", "category": "network", "deprecated": false},
			},
		},
		{
			name:     "typed string slice",
			expr:     `_.startsWith("prod-")`,
			data:     []string{"prod-api", "dev-api", "prod-web"},
			expected: []any{"prod-api", "prod-web"},
		},
		{
			name:     "[]any containing structs and pointers",
			expr:     `_.name == "http"`,
			data:     []any{Summary{Name: "http"}, &Summary{Name: "exec"}},
			expected: []any{map[string]any{"name": "http", "category": "", "deprecated": false}},
		},
		{
			name:     "typed nil []Summary returns empty non-nil slice",
			expr:     `_.name == "http"`,
			data:     nilSummarySlice,
			expected: []any{},
		},
		{
			name:     "typed nil []*Summary returns empty non-nil slice",
			expr:     `_.name == "http"`,
			data:     nilPtrSlice,
			expected: []any{},
		},
		{
			name:     "typed nil []any returns empty non-nil slice",
			expr:     `_.name == "http"`,
			data:     nilAnySlice,
			expected: []any{},
		},
		{
			name:        "[]byte rejected as ErrNotList",
			expr:        `_ > 0`,
			data:        []byte("hello"),
			wantNotList: true,
		},
		{
			name:        "nil []byte rejected as ErrNotList",
			expr:        `_ > 0`,
			data:        nilByteSlice,
			wantNotList: true,
		},
		{
			name:        "named byte slice rejected as ErrNotList",
			expr:        `_ > 0`,
			data:        rawBytes(`{"a":1}`),
			wantNotList: true,
		},
		{
			name:        "map rejected as ErrNotList",
			expr:        `_.ok`,
			data:        map[string]any{"ok": true},
			wantNotList: true,
		},
		{
			name:        "struct rejected as ErrNotList",
			expr:        `_.name == "http"`,
			data:        Summary{Name: "http"},
			wantNotList: true,
		},
		{
			name:        "scalar string rejected as ErrNotList",
			expr:        `_ == "solo"`,
			data:        "solo",
			wantNotList: true,
		},
		{
			name:        "nil interface rejected as ErrNotList",
			expr:        `_ == null`,
			data:        nil,
			wantNotList: true,
		},
		{
			name:          "invalid CEL syntax on typed slice",
			expr:          `_.name ==`,
			data:          []Summary{{Name: "http"}},
			wantErrSubstr: "where filter compilation error",
		},
		{
			name: "struct slice with unexported embedded struct promotes exported fields",
			expr: `_.name == "http"`,
			data: []EmbeddedUnexportedSummary{
				{unexportedBase: unexportedBase{Name: "http"}, Category: "network"},
				{unexportedBase: unexportedBase{Name: "exec"}, Category: "system"},
			},
			expected: []any{
				map[string]any{"name": "http", "category": "network"},
			},
		},
		{
			name:          "unnormalizable slice element surfaces normalization error (not ErrNotList)",
			expr:          `true`,
			data:          []UnnormalizableItem{{Callback: func() {}}},
			wantErrSubstr: "where filter input normalization error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := eval.EvaluateWhere(tt.expr, tt.data)
			if tt.wantNotList {
				require.Error(t, err)
				assert.True(t, errors.Is(err, ErrNotList), "expected ErrNotList, got: %v", err)
				assert.Nil(t, got)
				return
			}
			if tt.wantErrSubstr != "" {
				require.Error(t, err)
				assert.False(t, errors.Is(err, ErrNotList))
				assert.Contains(t, err.Error(), tt.wantErrSubstr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.expected, got)
			if len(tt.expected) == 0 {
				encoded, mErr := json.Marshal(got)
				require.NoError(t, mErr)
				assert.Equal(t, "[]", string(encoded))
			}
		})
	}
}

func BenchmarkEvaluateWhere_TypedStructSlice(b *testing.B) {
	type Summary struct {
		Name       string `json:"name" yaml:"name"`
		Category   string `json:"category" yaml:"category"`
		Deprecated bool   `json:"deprecated" yaml:"deprecated"`
	}

	eval, err := NewEvaluator()
	require.NoError(b, err)

	items := make([]Summary, 1000)
	for i := range items {
		if i%2 == 0 {
			items[i] = Summary{Name: "http", Category: "network"}
		} else {
			items[i] = Summary{Name: "exec", Category: "system", Deprecated: true}
		}
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, err := eval.EvaluateWhere(`_.name == "http" && !_.deprecated`, items)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestEvaluateWhere_ConcurrentRace(t *testing.T) {
	type Summary struct {
		Name string `json:"name"`
	}
	eval, err := NewEvaluator()
	require.NoError(t, err)

	items := []Summary{{Name: "http"}, {Name: "exec"}}
	var wg sync.WaitGroup
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 25 {
				got, evalErr := eval.EvaluateWhere(`_.name == "http"`, items)
				assert.NoError(t, evalErr)
				assert.Len(t, got, 1)
			}
		}()
	}
	wg.Wait()
}

func TestEvaluateWhere_NilMapSliceElementParity(t *testing.T) {
	eval, err := NewEvaluator()
	require.NoError(t, err)

	gotMapSlice, err := eval.EvaluateWhere("true", []map[string]any{nil, {"k": "v"}})
	require.NoError(t, err)

	gotAnySlice, err := eval.EvaluateWhere("true", []any{map[string]any(nil), map[string]any{"k": "v"}})
	require.NoError(t, err)

	require.Len(t, gotMapSlice, 2)
	assert.True(t, gotMapSlice[0] == nil, "nil map element in []map[string]any must normalize to untyped nil")
	assert.Equal(t, gotAnySlice, gotMapSlice)
}
