package navigator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectShape(t *testing.T) {
	tests := []struct {
		name     string
		data     any
		wantKind ShapeKind
		wantLen  int
	}{
		{
			name:     "nil",
			data:     nil,
			wantKind: ShapeScalar,
			wantLen:  0,
		},
		{
			name:     "string scalar",
			data:     "hello",
			wantKind: ShapeScalar,
			wantLen:  0,
		},
		{
			name:     "int scalar",
			data:     42,
			wantKind: ShapeScalar,
			wantLen:  0,
		},
		{
			name:     "empty map",
			data:     map[string]any{},
			wantKind: ShapeMap,
			wantLen:  0,
		},
		{
			name:     "map with values",
			data:     map[string]any{"a": 1, "b": 2},
			wantKind: ShapeMap,
			wantLen:  2,
		},
		{
			name:     "empty array",
			data:     []any{},
			wantKind: ShapeArray,
			wantLen:  0,
		},
		{
			name:     "simple array",
			data:     []any{1, 2, 3},
			wantKind: ShapeArray,
			wantLen:  3,
		},
		{
			name: "homogeneous array of objects",
			data: []any{
				map[string]any{"name": "Alice", "age": 30},
				map[string]any{"name": "Bob", "age": 25},
			},
			wantKind: ShapeHomogeneousArray,
			wantLen:  2,
		},
		{
			name: "heterogeneous array - different keys",
			data: []any{
				map[string]any{"name": "Alice"},
				map[string]any{"title": "Engineer"},
			},
			wantKind: ShapeArray,
			wantLen:  2,
		},
		{
			name: "mixed array - objects and scalars",
			data: []any{
				map[string]any{"name": "Alice"},
				"scalar",
			},
			wantKind: ShapeArray,
			wantLen:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shape := DetectShape(tt.data)
			assert.Equal(t, tt.wantKind, shape.Kind)
			assert.Equal(t, tt.wantLen, shape.Length)
		})
	}
}

func TestIsHomogeneousArray(t *testing.T) {
	tests := []struct {
		name       string
		data       any
		wantResult bool
		wantFields []string
	}{
		{
			name:       "nil",
			data:       nil,
			wantResult: false,
		},
		{
			name:       "not an array",
			data:       map[string]any{"a": 1},
			wantResult: false,
		},
		{
			name:       "empty array",
			data:       []any{},
			wantResult: false,
		},
		{
			name:       "array of scalars",
			data:       []any{1, 2, 3},
			wantResult: false,
		},
		{
			name: "single object",
			data: []any{
				map[string]any{"name": "Alice", "age": 30},
			},
			wantResult: true,
			wantFields: []string{"age", "name"},
		},
		{
			name: "homogeneous - same keys",
			data: []any{
				map[string]any{"id": 1, "name": "Alice"},
				map[string]any{"id": 2, "name": "Bob"},
				map[string]any{"id": 3, "name": "Charlie"},
			},
			wantResult: true,
			wantFields: []string{"id", "name"},
		},
		{
			name: "heterogeneous - different keys",
			data: []any{
				map[string]any{"id": 1, "name": "Alice"},
				map[string]any{"id": 2, "email": "bob@example.com"},
			},
			wantResult: false,
		},
		{
			name: "heterogeneous - extra key in one",
			data: []any{
				map[string]any{"id": 1, "name": "Alice"},
				map[string]any{"id": 2, "name": "Bob", "extra": true},
			},
			wantResult: false,
		},
		{
			name: "uses map[string]interface{} type",
			data: []interface{}{
				map[string]interface{}{"x": 1, "y": 2},
				map[string]interface{}{"x": 3, "y": 4},
			},
			wantResult: true,
			wantFields: []string{"x", "y"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, fields := IsHomogeneousArray(tt.data)
			assert.Equal(t, tt.wantResult, result)
			if tt.wantResult {
				assert.Equal(t, tt.wantFields, fields)
			}
		})
	}
}

func TestExtractColumnarData(t *testing.T) {
	t.Run("homogeneous array", func(t *testing.T) {
		data := []any{
			map[string]any{"name": "Alice", "age": 30},
			map[string]any{"name": "Bob", "age": 25},
		}

		columns, rows := ExtractColumnarData(data, nil)
		require.NotNil(t, columns)
		require.NotNil(t, rows)

		assert.Equal(t, []string{"age", "name"}, columns)
		assert.Len(t, rows, 2)
		// Values are stringified
		assert.Equal(t, "30", rows[0][0])
		assert.Equal(t, "Alice", rows[0][1])
		assert.Equal(t, "25", rows[1][0])
		assert.Equal(t, "Bob", rows[1][1])
	})

	t.Run("with custom field order", func(t *testing.T) {
		data := []any{
			map[string]any{"id": 1, "name": "Alice", "age": 30},
			map[string]any{"id": 2, "name": "Bob", "age": 25},
		}

		columns, rows := ExtractColumnarData(data, []string{"name", "id"})
		require.NotNil(t, columns)

		// Custom order with remaining fields appended
		assert.Equal(t, []string{"name", "id", "age"}, columns)
		assert.Equal(t, "Alice", rows[0][0])
		assert.Equal(t, "1", rows[0][1])
	})

	t.Run("non-homogeneous returns nil", func(t *testing.T) {
		data := []any{1, 2, 3}
		columns, rows := ExtractColumnarData(data, nil)
		assert.Nil(t, columns)
		assert.Nil(t, rows)
	})
}

func TestToStringKeyMap(t *testing.T) {
	tests := []struct {
		name   string
		input  any
		wantOk bool
	}{
		{"nil", nil, false},
		{"string", "hello", false},
		{"int", 42, false},
		{"map[string]any", map[string]any{"a": 1}, true},
		{"map[string]interface{}", map[string]interface{}{"a": 1}, true},
		{"slice", []any{1, 2}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := toStringKeyMap(tt.input)
			assert.Equal(t, tt.wantOk, ok)
		})
	}
}
