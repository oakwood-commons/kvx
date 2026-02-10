package loader

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Standard test JWT: {"alg":"HS256","typ":"JWT"}.{"sub":"1234567890","name":"John Doe","iat":1516239022}.<signature>
const validJWT = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

func TestIsJWT(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid JWT",
			input:    validJWT,
			expected: true,
		},
		{
			name:     "JWT with Bearer prefix",
			input:    "Bearer " + validJWT,
			expected: true,
		},
		{
			name:     "JWT with trailing newline",
			input:    validJWT + "\n",
			expected: true,
		},
		{
			name:     "JWT with leading/trailing whitespace",
			input:    "  " + validJWT + "  \n",
			expected: true,
		},
		{
			name:     "two-part string",
			input:    "part1.part2",
			expected: false,
		},
		{
			name:     "four-part string",
			input:    "part1.part2.part3.part4",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "regular JSON object",
			input:    `{"foo": "bar"}`,
			expected: false,
		},
		{
			name:     "regular YAML",
			input:    "foo: bar\nbaz: qux",
			expected: false,
		},
		{
			name:     "invalid base64url in header",
			input:    "not-valid-base64!.eyJzdWIiOiIxMjM0NTY3ODkwIn0.signature",
			expected: false,
		},
		{
			name:     "valid base64url but not JSON in header",
			input:    "aGVsbG8gd29ybGQ.eyJzdWIiOiIxMjM0NTY3ODkwIn0.signature", // "hello world" base64
			expected: false,
		},
		{
			name:     "empty parts",
			input:    "...",
			expected: false,
		},
		{
			name:     "one empty part",
			input:    "eyJhbGciOiJIUzI1NiJ9..sig",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsJWT(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDecodeJWT(t *testing.T) {
	t.Run("valid JWT returns correct header", func(t *testing.T) {
		result, err := DecodeJWT(validJWT)
		require.NoError(t, err)

		header, ok := result["header"].(map[string]any)
		require.True(t, ok, "header should be a map")
		assert.Equal(t, "HS256", header["alg"])
		assert.Equal(t, "JWT", header["typ"])
	})

	t.Run("valid JWT returns correct payload", func(t *testing.T) {
		result, err := DecodeJWT(validJWT)
		require.NoError(t, err)

		payload, ok := result["payload"].(map[string]any)
		require.True(t, ok, "payload should be a map")
		assert.Equal(t, "1234567890", payload["sub"])
		assert.Equal(t, "John Doe", payload["name"])
		assert.Equal(t, float64(1516239022), payload["iat"]) // JSON numbers are float64
	})

	t.Run("valid JWT returns signature as string", func(t *testing.T) {
		result, err := DecodeJWT(validJWT)
		require.NoError(t, err)

		sig, ok := result["signature"].(string)
		require.True(t, ok, "signature should be a string")
		assert.Equal(t, "SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c", sig)
	})

	t.Run("JWT with nested objects in payload", func(t *testing.T) {
		// {"alg":"HS256"}.{"user":{"id":123,"roles":["admin","user"]}}.sig
		// Header: eyJhbGciOiJIUzI1NiJ9
		// Payload: eyJ1c2VyIjp7ImlkIjoxMjMsInJvbGVzIjpbImFkbWluIiwidXNlciJdfX0
		jwt := "eyJhbGciOiJIUzI1NiJ9.eyJ1c2VyIjp7ImlkIjoxMjMsInJvbGVzIjpbImFkbWluIiwidXNlciJdfX0.c2ln"
		result, err := DecodeJWT(jwt)
		require.NoError(t, err)

		payload, ok := result["payload"].(map[string]any)
		require.True(t, ok)

		user, ok := payload["user"].(map[string]any)
		require.True(t, ok, "user should be a nested map")
		assert.Equal(t, float64(123), user["id"])

		roles, ok := user["roles"].([]any)
		require.True(t, ok, "roles should be an array")
		assert.Equal(t, []any{"admin", "user"}, roles)
	})

	t.Run("JWT with array claims", func(t *testing.T) {
		// {"alg":"HS256"}.{"aud":["api","web"]}.sig
		// Payload: eyJhdWQiOlsiYXBpIiwid2ViIl19
		jwt := "eyJhbGciOiJIUzI1NiJ9.eyJhdWQiOlsiYXBpIiwid2ViIl19.c2ln"
		result, err := DecodeJWT(jwt)
		require.NoError(t, err)

		payload, ok := result["payload"].(map[string]any)
		require.True(t, ok)

		aud, ok := payload["aud"].([]any)
		require.True(t, ok, "aud should be an array")
		assert.Equal(t, []any{"api", "web"}, aud)
	})

	t.Run("JWT with Bearer prefix", func(t *testing.T) {
		result, err := DecodeJWT("Bearer " + validJWT)
		require.NoError(t, err)
		assert.NotNil(t, result["header"])
		assert.NotNil(t, result["payload"])
	})

	t.Run("JWT with whitespace", func(t *testing.T) {
		result, err := DecodeJWT("  " + validJWT + "  \n")
		require.NoError(t, err)
		assert.NotNil(t, result["header"])
	})

	t.Run("invalid JWT with wrong number of parts", func(t *testing.T) {
		_, err := DecodeJWT("only.two")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected 3 parts")
	})

	t.Run("invalid base64url in header", func(t *testing.T) {
		_, err := DecodeJWT("not-valid!!!.eyJzdWIiOiIxIn0.sig")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid JWT header")
	})

	t.Run("invalid JSON in header", func(t *testing.T) {
		// "hello" in base64url - valid base64 but not JSON object
		_, err := DecodeJWT("aGVsbG8.eyJzdWIiOiIxIn0.sig")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid JWT header JSON")
	})

	t.Run("invalid base64url in payload", func(t *testing.T) {
		_, err := DecodeJWT("eyJhbGciOiJIUzI1NiJ9.not-valid!!!.sig")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid JWT payload")
	})

	t.Run("invalid JSON in payload", func(t *testing.T) {
		// valid header, but payload is just "hello" not JSON
		_, err := DecodeJWT("eyJhbGciOiJIUzI1NiJ9.aGVsbG8.sig")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid JWT payload JSON")
	})
}

func TestLoadJWT(t *testing.T) {
	t.Run("returns slice with one element", func(t *testing.T) {
		result, err := loadJWT(validJWT)
		require.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("element is a map with header, payload, signature", func(t *testing.T) {
		result, err := loadJWT(validJWT)
		require.NoError(t, err)

		m, ok := result[0].(map[string]any)
		require.True(t, ok)
		assert.Contains(t, m, "header")
		assert.Contains(t, m, "payload")
		assert.Contains(t, m, "signature")
	})

	t.Run("propagates decode errors", func(t *testing.T) {
		_, err := loadJWT("invalid")
		require.Error(t, err)
	})
}
