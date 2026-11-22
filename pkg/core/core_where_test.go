package core

import (
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type providerSummary struct {
	Name         string `json:"name" yaml:"name"`
	DisplayName  string `json:"displayName" yaml:"displayName"`
	Category     string `json:"category" yaml:"category"`
	Capabilities string `json:"capabilities" yaml:"capabilities"`
	Deprecated   bool   `json:"deprecated" yaml:"deprecated"`
}

type yamlTaggedItem struct {
	ServiceName string `yaml:"service_name"`
	Port        int    `yaml:"port"`
}

func TestEngineFilterWhere(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	var nilSummarySlice []providerSummary
	var nilPtrSlice []*providerSummary
	var nilAnySlice []any
	var nilBytes []byte

	tests := []struct {
		name          string
		expr          string
		root          any
		expected      []any
		wantNotList   bool
		wantErrSubstr string
	}{
		{
			name: "typed struct slice []Summary matches identically to map slice",
			expr: `_.name == "http"`,
			root: []providerSummary{
				{Name: "http", DisplayName: "HTTP", Category: "network", Capabilities: "from,action"},
				{Name: "exec", DisplayName: "Exec", Category: "system", Capabilities: "action"},
			},
			expected: []any{
				map[string]any{
					"name":         "http",
					"displayName":  "HTTP",
					"category":     "network",
					"capabilities": "from,action",
					"deprecated":   false,
				},
			},
		},
		{
			name: "pointer-to-struct slice []*Summary matches identically",
			expr: `_.name == "http"`,
			root: []*providerSummary{
				{Name: "http", DisplayName: "HTTP", Category: "network", Capabilities: "from,action"},
				{Name: "exec", DisplayName: "Exec", Category: "system", Capabilities: "action"},
				nil,
			},
			expected: []any{
				map[string]any{
					"name":         "http",
					"displayName":  "HTTP",
					"category":     "network",
					"capabilities": "from,action",
					"deprecated":   false,
				},
			},
		},
		{
			name: "map slice []map[string]any",
			expr: `_.name == "http"`,
			root: []map[string]any{
				{"name": "http", "displayName": "HTTP", "category": "network", "capabilities": "from,action", "deprecated": false},
				{"name": "exec", "displayName": "Exec", "category": "system", "capabilities": "action", "deprecated": false},
			},
			expected: []any{
				map[string]any{
					"name":         "http",
					"displayName":  "HTTP",
					"category":     "network",
					"capabilities": "from,action",
					"deprecated":   false,
				},
			},
		},
		{
			name: "yaml-tagged struct slice",
			expr: `_.service_name == "api" && _.port == 8080`,
			root: []yamlTaggedItem{
				{ServiceName: "api", Port: 8080},
				{ServiceName: "worker", Port: 9090},
			},
			expected: []any{
				map[string]any{"service_name": "api", "port": 8080},
			},
		},
		{
			name: "[]any with mixed struct, pointer, and map items",
			expr: `_.name == "http"`,
			root: []any{
				providerSummary{Name: "http", DisplayName: "HTTP", Category: "network", Capabilities: "from"},
				&providerSummary{Name: "exec", DisplayName: "Exec", Category: "system", Capabilities: "action"},
				map[string]any{"name": "cel"},
			},
			expected: []any{
				map[string]any{
					"name":         "http",
					"displayName":  "HTTP",
					"category":     "network",
					"capabilities": "from",
					"deprecated":   false,
				},
			},
		},
		{
			name: "zero matches on typed slice returns non-nil []any{} marshaling to []",
			expr: `_.name == "nonexistent"`,
			root: []providerSummary{
				{Name: "http"},
				{Name: "exec"},
			},
			expected: []any{},
		},
		{
			name:     "typed nil []Summary returns non-nil []any{} marshaling to []",
			expr:     `_.name == "http"`,
			root:     nilSummarySlice,
			expected: []any{},
		},
		{
			name:     "typed nil []*Summary returns non-nil []any{} marshaling to []",
			expr:     `_.name == "http"`,
			root:     nilPtrSlice,
			expected: []any{},
		},
		{
			name:     "typed nil []any returns non-nil []any{} marshaling to []",
			expr:     `_.name == "http"`,
			root:     nilAnySlice,
			expected: []any{},
		},
		{
			name:     "empty []Summary{} returns non-nil []any{} marshaling to []",
			expr:     `_.name == "http"`,
			root:     []providerSummary{},
			expected: []any{},
		},
		{
			name:        "[]byte returns ErrNotList",
			expr:        `_ > 0`,
			root:        []byte("binary-payload"),
			wantNotList: true,
		},
		{
			name:        "typed nil []byte returns ErrNotList",
			expr:        `_ > 0`,
			root:        nilBytes,
			wantNotList: true,
		},
		{
			name:        "json.RawMessage returns ErrNotList",
			expr:        `_.ok`,
			root:        json.RawMessage(`{"ok":true}`),
			wantNotList: true,
		},
		{
			name:        "map[string]any returns ErrNotList",
			expr:        `_.ok`,
			root:        map[string]any{"name": "solo", "ok": true},
			wantNotList: true,
		},
		{
			name:        "struct returns ErrNotList",
			expr:        `_.name == "http"`,
			root:        providerSummary{Name: "http"},
			wantNotList: true,
		},
		{
			name:        "pointer to struct returns ErrNotList",
			expr:        `_.name == "http"`,
			root:        &providerSummary{Name: "http"},
			wantNotList: true,
		},
		{
			name:        "string scalar returns ErrNotList",
			expr:        `_ == "solo"`,
			root:        "solo",
			wantNotList: true,
		},
		{
			name:        "int scalar returns ErrNotList",
			expr:        `_ > 0`,
			root:        42,
			wantNotList: true,
		},
		{
			name:        "bool scalar returns ErrNotList",
			expr:        `_ == true`,
			root:        true,
			wantNotList: true,
		},
		{
			name:        "nil returns ErrNotList",
			expr:        `_ == null`,
			root:        nil,
			wantNotList: true,
		},
		{
			name:          "invalid CEL syntax returns compilation error (not ErrNotList)",
			expr:          `_.name ==`,
			root:          []providerSummary{{Name: "http"}},
			wantErrSubstr: "where filter compilation error",
		},
		{
			name:          "compile-time non-boolean CEL expression returns error",
			expr:          `size(_)`,
			root:          []providerSummary{{Name: "http"}},
			wantErrSubstr: "where filter expression must return a boolean",
		},
		{
			name:          "runtime non-boolean CEL expression returns error",
			expr:          `_.name`,
			root:          []providerSummary{{Name: "http"}},
			wantErrSubstr: "where filter expression must return a boolean",
		},
		{
			name:          "invalid CEL expression on typed nil slice still errors",
			expr:          `_.name + "suffix"`,
			root:          nilSummarySlice,
			wantErrSubstr: "where filter expression must return a boolean",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.FilterWhere(tt.expr, tt.root)
			evalGot, evalErr := engine.EvaluateWhere(tt.expr, tt.root)

			if tt.wantNotList {
				require.Error(t, err)
				assert.True(t, errors.Is(err, ErrNotList), "FilterWhere error should wrap ErrNotList, got: %v", err)
				assert.True(t, got == nil, "FilterWhere error return value must be untyped nil, got %T", got)

				require.Error(t, evalErr)
				assert.True(t, errors.Is(evalErr, ErrNotList), "EvaluateWhere error should wrap ErrNotList, got: %v", evalErr)
				assert.Nil(t, evalGot)
				return
			}

			if tt.wantErrSubstr != "" {
				require.Error(t, err)
				assert.False(t, errors.Is(err, ErrNotList))
				assert.Contains(t, err.Error(), tt.wantErrSubstr)
				assert.True(t, got == nil, "FilterWhere error return value must be untyped nil, got %T", got)

				require.Error(t, evalErr)
				assert.False(t, errors.Is(evalErr, ErrNotList))
				assert.Contains(t, evalErr.Error(), tt.wantErrSubstr)
				return
			}

			require.NoError(t, err)
			require.NoError(t, evalErr)

			slice, ok := got.([]any)
			require.True(t, ok, "FilterWhere result type = %T, want []any", got)
			require.NotNil(t, slice)
			assert.Equal(t, tt.expected, slice)
			assert.Equal(t, tt.expected, evalGot)

			encoded, mErr := json.Marshal(got)
			require.NoError(t, mErr)
			if len(tt.expected) == 0 {
				assert.Equal(t, "[]", string(encoded), "empty result must marshal to [] and never null")
			} else {
				assert.NotEqual(t, "null", string(encoded))
			}
		})
	}
}

func TestEngineFilterWhere_EdgeEvaluators(t *testing.T) {
	t.Run("nil engine errors", func(t *testing.T) {
		var eng *Engine
		got, err := eng.FilterWhere(`_.ok`, []any{})
		require.Error(t, err)
		assert.True(t, got == nil, "FilterWhere error return value must be untyped nil, got %T", got)
	})

	t.Run("evaluator without WhereEvaluator errors", func(t *testing.T) {
		eng := &Engine{Evaluator: stubPlainEvaluator{}}
		got, err := eng.FilterWhere(`_.ok`, []any{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not support where filtering")
		assert.True(t, got == nil, "FilterWhere error return value must be untyped nil, got %T", got)
	})

	t.Run("custom WhereEvaluator returning nil slice normalized to []any{}", func(t *testing.T) {
		eng := &Engine{Evaluator: stubNilSliceWhereEvaluator{}}
		got, err := eng.FilterWhere(`_.ok`, []any{})
		require.NoError(t, err)
		slice, ok := got.([]any)
		require.True(t, ok)
		require.NotNil(t, slice)
		encoded, mErr := json.Marshal(got)
		require.NoError(t, mErr)
		assert.Equal(t, "[]", string(encoded))
	})
}

type stubPlainEvaluator struct{}

func (stubPlainEvaluator) Evaluate(_ string, root any) (any, error) {
	return root, nil
}

type stubNilSliceWhereEvaluator struct{}

func (stubNilSliceWhereEvaluator) Evaluate(_ string, root any) (any, error) {
	return root, nil
}

func (stubNilSliceWhereEvaluator) EvaluateWhere(_ string, _ any) ([]any, error) {
	return nil, nil
}

func TestEngineFilterWhere_ConcurrentRace(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	structSlice := []providerSummary{
		{Name: "http", Category: "network"},
		{Name: "exec", Category: "system"},
	}
	ptrSlice := []*providerSummary{
		{Name: "http", Category: "network"},
		{Name: "cel", Category: "transform"},
	}
	nonList := map[string]any{"name": "solo"}

	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 25 {
				res1, err1 := engine.FilterWhere(`_.name == "http"`, structSlice)
				assert.NoError(t, err1)
				assert.Len(t, res1, 1)

				res2, err2 := engine.FilterWhere(`_.name == "cel"`, ptrSlice)
				assert.NoError(t, err2)
				assert.Len(t, res2, 1)

				_, err3 := engine.FilterWhere(`_.name == "solo"`, nonList)
				assert.True(t, errors.Is(err3, ErrNotList))
			}
		}()
	}
	wg.Wait()
}

func BenchmarkEngineFilterWhere(b *testing.B) {
	engine, err := New()
	require.NoError(b, err)

	const size = 500
	structs := make([]providerSummary, size)
	ptrs := make([]*providerSummary, size)
	maps := make([]map[string]any, size)

	for i := range size {
		name := "exec"
		if i%5 == 0 {
			name = "http"
		}
		s := providerSummary{
			Name:         name,
			DisplayName:  "Provider",
			Category:     "network",
			Capabilities: "from,action",
		}
		structs[i] = s
		ptrs[i] = &s
		maps[i] = map[string]any{
			"name":         name,
			"displayName":  "Provider",
			"category":     "network",
			"capabilities": "from,action",
			"deprecated":   false,
		}
	}

	b.Run("typed_struct_slice", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			if _, err := engine.FilterWhere(`_.name == "http"`, structs); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("pointer_struct_slice", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			if _, err := engine.FilterWhere(`_.name == "http"`, ptrs); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("map_slice", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for b.Loop() {
			if _, err := engine.FilterWhere(`_.name == "http"`, maps); err != nil {
				b.Fatal(err)
			}
		}
	})
}
