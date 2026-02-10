package limiter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid limit only",
			cfg:     Config{Limit: 10},
			wantErr: false,
		},
		{
			name:    "valid offset only",
			cfg:     Config{Offset: 5},
			wantErr: false,
		},
		{
			name:    "valid limit and offset",
			cfg:     Config{Limit: 10, Offset: 5},
			wantErr: false,
		},
		{
			name:    "valid tail only",
			cfg:     Config{Tail: 10},
			wantErr: false,
		},
		{
			name:    "tail ignores offset (valid)",
			cfg:     Config{Tail: 10, Offset: 5},
			wantErr: false,
		},
		{
			name:    "limit and tail mutually exclusive",
			cfg:     Config{Limit: 10, Tail: 5},
			wantErr: true,
			errMsg:  "mutually exclusive",
		},
		{
			name:    "negative limit invalid",
			cfg:     Config{Limit: -1},
			wantErr: true,
			errMsg:  "non-negative",
		},
		{
			name:    "negative offset invalid",
			cfg:     Config{Offset: -1},
			wantErr: true,
			errMsg:  "non-negative",
		},
		{
			name:    "negative tail invalid",
			cfg:     Config{Tail: -1},
			wantErr: true,
			errMsg:  "non-negative",
		},
		{
			name:    "zero values valid",
			cfg:     Config{Limit: 0, Offset: 0, Tail: 0},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfigIsActive(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		wantBool bool
	}{
		{
			name:     "no flags set",
			cfg:      Config{},
			wantBool: false,
		},
		{
			name:     "limit set",
			cfg:      Config{Limit: 10},
			wantBool: true,
		},
		{
			name:     "offset set",
			cfg:      Config{Offset: 5},
			wantBool: true,
		},
		{
			name:     "tail set",
			cfg:      Config{Tail: 10},
			wantBool: true,
		},
		{
			name:     "all flags set",
			cfg:      Config{Limit: 10, Offset: 5, Tail: 0}, // tail not really set
			wantBool: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.IsActive()
			assert.Equal(t, tt.wantBool, got)
		})
	}
}

func TestApplyToArray(t *testing.T) {
	arr := []interface{}{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	tests := []struct {
		name string
		cfg  Config
		want []interface{}
	}{
		{
			name: "limit only",
			cfg:  Config{Limit: 3},
			want: []interface{}{1, 2, 3},
		},
		{
			name: "offset only",
			cfg:  Config{Offset: 5},
			want: []interface{}{6, 7, 8, 9, 10},
		},
		{
			name: "limit and offset",
			cfg:  Config{Limit: 3, Offset: 2},
			want: []interface{}{3, 4, 5},
		},
		{
			name: "tail only",
			cfg:  Config{Tail: 3},
			want: []interface{}{8, 9, 10},
		},
		{
			name: "offset larger than array",
			cfg:  Config{Offset: 20},
			want: []interface{}{},
		},
		{
			name: "limit larger than remaining",
			cfg:  Config{Limit: 100, Offset: 5},
			want: []interface{}{6, 7, 8, 9, 10},
		},
		{
			name: "tail larger than array",
			cfg:  Config{Tail: 100},
			want: []interface{}{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
		{
			name: "limit zero (unlimited)",
			cfg:  Config{Limit: 0},
			want: []interface{}{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cfg.Apply(arr)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestApplyToMap(t *testing.T) {
	m := map[string]interface{}{
		"a": 1,
		"b": 2,
		"c": 3,
		"d": 4,
		"e": 5,
	}

	tests := []struct {
		name  string
		cfg   Config
		check func(t *testing.T, result interface{})
	}{
		{
			name: "limit only",
			cfg:  Config{Limit: 2},
			check: func(t *testing.T, result interface{}) {
				resultMap := result.(map[string]interface{})
				assert.Len(t, resultMap, 2)
				// Keys should be sorted: a, b
				assert.Equal(t, 1, resultMap["a"])
				assert.Equal(t, 2, resultMap["b"])
			},
		},
		{
			name: "offset only",
			cfg:  Config{Offset: 2},
			check: func(t *testing.T, result interface{}) {
				resultMap := result.(map[string]interface{})
				assert.Len(t, resultMap, 3)
				// Keys should be c, d, e (after skipping a, b)
				assert.Equal(t, 3, resultMap["c"])
				assert.Equal(t, 4, resultMap["d"])
				assert.Equal(t, 5, resultMap["e"])
			},
		},
		{
			name: "limit and offset",
			cfg:  Config{Limit: 2, Offset: 1},
			check: func(t *testing.T, result interface{}) {
				resultMap := result.(map[string]interface{})
				assert.Len(t, resultMap, 2)
				// Keys should be b, c
				assert.Equal(t, 2, resultMap["b"])
				assert.Equal(t, 3, resultMap["c"])
			},
		},
		{
			name: "tail only",
			cfg:  Config{Tail: 2},
			check: func(t *testing.T, result interface{}) {
				resultMap := result.(map[string]interface{})
				assert.Len(t, resultMap, 2)
				// Last 2 keys: d, e
				assert.Equal(t, 4, resultMap["d"])
				assert.Equal(t, 5, resultMap["e"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cfg.Apply(m)
			tt.check(t, result)
		})
	}
}

func TestApplyToScalar(t *testing.T) {
	tests := []struct {
		name string
		data interface{}
		cfg  Config
		want interface{}
	}{
		{
			name: "string scalar",
			data: "hello",
			cfg:  Config{Limit: 10},
			want: "hello",
		},
		{
			name: "number scalar",
			data: 42,
			cfg:  Config{Limit: 10},
			want: 42,
		},
		{
			name: "nil",
			data: nil,
			cfg:  Config{Limit: 10},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cfg.Apply(tt.data)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestApplyEdgeCases(t *testing.T) {
	t.Run("empty array", func(t *testing.T) {
		result := Config{Limit: 10}.Apply([]interface{}{})
		assert.Equal(t, []interface{}{}, result)
	})

	t.Run("empty map", func(t *testing.T) {
		result := Config{Limit: 10}.Apply(map[string]interface{}{})
		assert.Equal(t, map[string]interface{}{}, result)
	})

	t.Run("single element array with limit 1", func(t *testing.T) {
		arr := []interface{}{42}
		result := Config{Limit: 1}.Apply(arr)
		assert.Equal(t, []interface{}{42}, result)
	})

	t.Run("offset equals array length", func(t *testing.T) {
		arr := []interface{}{1, 2, 3}
		result := Config{Offset: 3}.Apply(arr)
		assert.Equal(t, []interface{}{}, result)
	})

	t.Run("tail zero is inactive", func(t *testing.T) {
		arr := []interface{}{1, 2, 3, 4, 5}
		result := Config{Tail: 0}.Apply(arr)
		assert.Equal(t, arr, result)
	})
}

func TestTailIgnoresOffset(t *testing.T) {
	arr := []interface{}{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	// Even though offset is set, it should be ignored when tail is used
	result := Config{Tail: 3, Offset: 5}.Apply(arr)
	assert.Equal(t, []interface{}{8, 9, 10}, result)
}
