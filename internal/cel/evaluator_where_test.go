package cel

import (
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
			name:    "non-boolean expression errors",
			expr:    `_.name`,
			data:    []interface{}{map[string]interface{}{"name": "x"}},
			wantErr: "where filter expression must return a boolean",
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

	b.ResetTimer()

	for b.Loop() {
		_, err := eval.EvaluateWhere(`_.active == true`, items)
		if err != nil {
			b.Fatal(err)
		}
	}
}
