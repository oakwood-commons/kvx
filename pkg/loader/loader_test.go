package loader

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
		wantErr bool
	}{
		{
			name:    "single object",
			input:   `{"name": "test", "value": 42}`,
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "single array",
			input:   `[1, 2, 3]`,
			wantLen: 1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadData(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantLen, len(got))
		})
	}

	t.Run("invalid JSON falls back to YAML", func(t *testing.T) {
		got, err := LoadData(`{invalid}`)
		require.NoError(t, err)
		require.Len(t, got, 1)
		// YAML parses {invalid} as a flow mapping with key "invalid" and nil value
		assert.Equal(t, map[string]interface{}{"invalid": nil}, got[0])
	})
}

func TestLoadYAML(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
		wantErr bool
	}{
		{
			name: "single YAML object",
			input: `name: test
value: 42`,
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "YAML with nested object",
			input: `person:
  name: Alice
  age: 30`,
			wantLen: 1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadData(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantLen, len(got))
		})
	}
}

func TestLoadMultiDocYAML(t *testing.T) {
	input := `name: Alice
age: 30
---
name: Bob
age: 25
---
name: Charlie
age: 35`

	got, err := LoadData(input)
	require.NoError(t, err)
	assert.Equal(t, 3, len(got))

	// Verify each document is a map
	for _, doc := range got {
		assert.IsType(t, map[string]interface{}{}, doc)
	}
}

func TestLoadNDJSON(t *testing.T) {
	input := `{"id": 1, "message": "first"}
{"id": 2, "message": "second"}
{"id": 3, "message": "third"}`

	got, err := LoadData(input)
	require.NoError(t, err)
	assert.Equal(t, 3, len(got))

	// Verify each item is a map
	for _, item := range got {
		assert.IsType(t, map[string]interface{}{}, item)
	}
}

func TestLoadNDJSONWithBlankLines(t *testing.T) {
	input := `{"id": 1, "name": "Alice"}

{"id": 2, "name": "Bob"}

{"id": 3, "name": "Charlie"}`

	got, err := LoadData(input)
	require.NoError(t, err)
	// Blank lines should be skipped
	assert.Equal(t, 3, len(got))
}

func TestLoadMixed(t *testing.T) {
	// Test that preference is: multi-doc YAML > NDJSON > single JSON/YAML
	t.Run("multi-doc YAML takes precedence", func(t *testing.T) {
		input := `name: Alice
---
name: Bob`
		got, err := LoadData(input)
		require.NoError(t, err)
		assert.Equal(t, 2, len(got))
	})

	t.Run("NDJSON without --- markers", func(t *testing.T) {
		input := `{"id": 1}
{"id": 2}
{"id": 3}`
		got, err := LoadData(input)
		require.NoError(t, err)
		assert.Equal(t, 3, len(got))
	})

	t.Run("single YAML document", func(t *testing.T) {
		input := `name: test
value: 42`
		got, err := LoadData(input)
		require.NoError(t, err)
		assert.Equal(t, 1, len(got))
	})
}

func TestLoadDataJWT(t *testing.T) {
	jwt := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

	t.Run("LoadData with JWT returns decoded structure", func(t *testing.T) {
		got, err := LoadData(jwt)
		require.NoError(t, err)
		assert.Len(t, got, 1)

		m, ok := got[0].(map[string]any)
		require.True(t, ok)
		assert.Contains(t, m, "header")
		assert.Contains(t, m, "payload")
		assert.Contains(t, m, "signature")
	})

	t.Run("LoadRoot with JWT returns single map", func(t *testing.T) {
		got, err := LoadRoot(jwt)
		require.NoError(t, err)

		m, ok := got.(map[string]any)
		require.True(t, ok, "LoadRoot should return map directly, not wrapped in slice")
		assert.Contains(t, m, "header")
	})

	t.Run("JWT with Bearer prefix via LoadData", func(t *testing.T) {
		got, err := LoadData("Bearer " + jwt)
		require.NoError(t, err)
		assert.Len(t, got, 1)
	})
}

func TestLoadTOML(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
		wantErr bool
	}{
		{
			name: "simple TOML with section",
			input: `[server]
host = "localhost"
port = 8080`,
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "TOML with nested tables",
			input: `[database]
host = "db.example.com"

[database.credentials]
username = "admin"`,
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "TOML array of tables",
			input: `[[users]]
name = "Alice"

[[users]]
name = "Bob"`,
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "key-value only TOML",
			input: `name = "test"
value = 42`,
			wantLen: 1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadData(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantLen, len(got))
		})
	}
}

func TestIsLikelyTOML(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name: "TOML with section header",
			input: `[server]
host = "localhost"`,
			want: true,
		},
		{
			name: "TOML with array of tables",
			input: `[[items]]
name = "item1"`,
			want: true,
		},
		{
			name: "key-value assignments",
			input: `name = "test"
value = 42
enabled = true`,
			want: true,
		},
		{
			name: "YAML syntax",
			input: `name: test
value: 42`,
			want: false,
		},
		{
			name:  "JSON object",
			input: `{"name": "test"}`,
			want:  false,
		},
		{
			name: "YAML list",
			input: `- item1
- item2`,
			want: false,
		},
		{
			name: "quoted key assignment",
			input: `"table name" = "value"
"another-key" = 42`,
			want: true,
		},
		{
			name: "dotted key assignment",
			input: `database.host = "localhost"
database.port = 5432`,
			want: true,
		},
		{
			name: "quoted section header",
			input: `["table name"]
key = "value"`,
			want: true,
		},
		{
			name: "dotted section header",
			input: `[database.credentials]
username = "admin"`,
			want: true,
		},
		{
			name: "mixed dotted and quoted section",
			input: `[server."host.name"]
value = "test"`,
			want: true,
		},
		{
			name:  "JSON array should not match",
			input: `[1, 2, 3]`,
			want:  false,
		},
		{
			name: "indented JSON-style array not mistaken for TOML section",
			input: `            - when: _.gcpArchitecture == "2.0"
              expression: |
                ["legacy"]`,
			want: false,
		},
		{
			name: "section header alone without key-value lines",
			input: `[server]
some text that is not a kv pair`,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isLikelyTOML(tt.input)
			assert.Equal(t, tt.want, got, "isLikelyTOML(%q)", tt.input)
		})
	}
}

func TestLoadDataTOMLIntegration(t *testing.T) {
	input := `title = "Sample"

[server]
host = "localhost"
port = 8080

[[users]]
name = "Alice"
roles = ["admin", "user"]

[[users]]
name = "Bob"
roles = ["user"]`

	got, err := LoadData(input)
	require.NoError(t, err)
	assert.Equal(t, 1, len(got))

	// Verify structure
	m, ok := got[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Sample", m["title"])
	assert.Contains(t, m, "server")
	assert.Contains(t, m, "users")

	// Verify nested server
	server, ok := m["server"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "localhost", server["host"])
	assert.Equal(t, int64(8080), server["port"])

	// Verify users array
	users, ok := m["users"].([]any)
	require.True(t, ok)
	assert.Len(t, users, 2)
}

func TestLoadDataEmpty(t *testing.T) {
	_, err := LoadData("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty input")
}

func TestLoadNDJSONWithPlainStrings(t *testing.T) {
	input := `{"id": 1, "message": "first"}
this is a plain string line
{"id": 2, "message": "second"}
another string
{"id": 3, "message": "third"}`

	got, err := LoadData(input)
	require.NoError(t, err)
	// Should have 5 items: 3 JSON objects + 2 plain strings
	assert.Equal(t, 5, len(got))

	// Verify types
	assert.IsType(t, map[string]interface{}{}, got[0])
	assert.IsType(t, "", got[1])
	assert.IsType(t, map[string]interface{}{}, got[2])
	assert.IsType(t, "", got[3])
	assert.IsType(t, map[string]interface{}{}, got[4])

	// Verify string values
	assert.Equal(t, "this is a plain string line", got[1])
	assert.Equal(t, "another string", got[3])
}

func TestLoadNDJSONWithCarriageReturns(t *testing.T) {
	t.Run("carriage return separates lines from CLI progress indicators", func(t *testing.T) {
		// CLI tools sometimes use \r to overwrite lines (progress indicators),
		// mixing with JSON log output
		input := "{\"level\":\"debug\"}\r❌ error message\n{\"level\":\"info\"}"
		got, err := LoadData(input)
		require.NoError(t, err)
		// Should parse as 3 items: 2 JSON objects + 1 string
		assert.Equal(t, 3, len(got))
		assert.IsType(t, map[string]interface{}{}, got[0])
		assert.IsType(t, "", got[1])
		assert.IsType(t, map[string]interface{}{}, got[2])
		assert.Equal(t, "❌ error message", got[1])
	})

	t.Run("Windows CRLF line endings", func(t *testing.T) {
		input := "{\"id\":1}\r\n{\"id\":2}\r\n{\"id\":3}"
		got, err := LoadData(input)
		require.NoError(t, err)
		assert.Equal(t, 3, len(got))
	})

	t.Run("mixed line endings", func(t *testing.T) {
		input := "{\"a\":1}\n{\"b\":2}\r\n{\"c\":3}\r{\"d\":4}"
		got, err := LoadData(input)
		require.NoError(t, err)
		assert.Equal(t, 4, len(got))
	})

	t.Run("file with .ndjson extension and carriage returns", func(t *testing.T) {
		// Ensure LoadFile (extension-based parsing) also normalizes CR
		tmpFile, err := os.CreateTemp("", "test-*.ndjson")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		content := "{\"level\":\"debug\"}\r❌ error message\n{\"level\":\"info\"}"
		_, err = tmpFile.WriteString(content)
		require.NoError(t, err)
		tmpFile.Close()

		got, err := LoadFile(tmpFile.Name())
		require.NoError(t, err)

		// Should parse as slice of 3 items
		gotSlice, ok := got.([]interface{})
		require.True(t, ok, "expected slice, got %T", got)
		assert.Equal(t, 3, len(gotSlice))
		assert.IsType(t, map[string]interface{}{}, gotSlice[0])
		assert.IsType(t, "", gotSlice[1])
		assert.IsType(t, map[string]interface{}{}, gotSlice[2])
	})
}

func TestLoadYAMLWithListItems(t *testing.T) {
	t.Run("YAML with many bare list items is not misdetected as NDJSON", func(t *testing.T) {
		input := `linters:
  enable:
    - asciicheck
    - bodyclose
    - dogsled
    - dupl
    - errcheck
    - errorlint
    - exhaustive
    - misspell
    - nakedret
    - prealloc`

		got, err := LoadData(input)
		require.NoError(t, err)
		assert.Equal(t, 1, len(got), "should parse as a single YAML document")
		assert.IsType(t, map[string]interface{}{}, got[0], "should be a map, not an array of strings")
	})

	t.Run("YAML map with nested arrays and few colons", func(t *testing.T) {
		input := `version: "2"
items:
  - alpha
  - bravo
  - charlie
  - delta
  - echo
  - foxtrot
  - golf
  - hotel`

		got, err := LoadData(input)
		require.NoError(t, err)
		assert.Equal(t, 1, len(got))

		m, ok := got[0].(map[string]interface{})
		require.True(t, ok, "should parse as a YAML map")
		assert.Equal(t, "2", m["version"])
	})

	t.Run("plain strings without JSON markers fall through to YAML", func(t *testing.T) {
		input := `hello world
foo bar
baz qux`

		got, err := LoadData(input)
		require.NoError(t, err)
		// Without JSON-like lines, this should not be NDJSON;
		// it falls through to single YAML (parsed as a string)
		assert.Equal(t, 1, len(got))
	})
}

func TestLoadRootSingle(t *testing.T) {
	root, err := LoadRoot(`{"name":"test"}`)
	require.NoError(t, err)
	assert.IsType(t, map[string]interface{}{}, root)
}

func TestLoadRootMulti(t *testing.T) {
	root, err := LoadRoot(`name: Alice
---
name: Bob`)
	require.NoError(t, err)
	arr, ok := root.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", root)
	}
	assert.Equal(t, 2, len(arr))
}

func TestLoadFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/data.yaml"
	content := []byte("name: test\nvalue: 42\n")
	require.NoError(t, os.WriteFile(path, content, 0o644))

	root, err := LoadFile(path)
	require.NoError(t, err)
	assert.IsType(t, map[string]interface{}{}, root)
}

func TestLoadFileHonorsExtension(t *testing.T) {
	dir := t.TempDir()

	t.Run("yaml extension parsed as YAML", func(t *testing.T) {
		path := dir + "/data.yml"
		require.NoError(t, os.WriteFile(path, []byte("name: test\nvalue: 42\n"), 0o644))

		root, err := LoadFile(path)
		require.NoError(t, err)
		m, ok := root.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "test", m["name"])
	})

	t.Run("json extension parsed as JSON", func(t *testing.T) {
		path := dir + "/data.json"
		require.NoError(t, os.WriteFile(path, []byte(`{"key":"val"}`), 0o644))

		root, err := LoadFile(path)
		require.NoError(t, err)
		m, ok := root.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "val", m["key"])
	})

	t.Run("toml extension parsed as TOML", func(t *testing.T) {
		path := dir + "/data.toml"
		require.NoError(t, os.WriteFile(path, []byte("[server]\nhost = \"localhost\"\n"), 0o644))

		root, err := LoadFile(path)
		require.NoError(t, err)
		m, ok := root.(map[string]interface{})
		require.True(t, ok)
		assert.Contains(t, m, "server")
	})
}

func TestLoadDataFallsThrough(t *testing.T) {
	t.Run("YAML with indented JSON arrays not misdetected as TOML", func(t *testing.T) {
		// This input has an indented ["legacy"] that previously triggered
		// a false-positive TOML detection, causing a parse failure.
		input := `items:
  - when: arch == "2.0"
    expression: |
      ["legacy"]
  - when: arch == "3.0"
    expression: |
      ["modern"]`

		got, err := LoadData(input)
		require.NoError(t, err)
		assert.Len(t, got, 1)
		assert.IsType(t, map[string]interface{}{}, got[0])
	})

	t.Run("wrong extension falls back to correct parser", func(t *testing.T) {
		// Write valid JSON content with a .toml extension — the TOML
		// parser should fail, then fallback should succeed with JSON.
		dir := t.TempDir()
		path := dir + "/oops.toml"
		require.NoError(t, os.WriteFile(path, []byte(`{"key":"val"}`), 0o644))

		root, err := LoadFile(path)
		require.NoError(t, err)
		m, ok := root.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "val", m["key"])
	})
}

func TestLoadObject(t *testing.T) {
	type sample struct {
		Name string
	}

	t.Run("nil interface", func(t *testing.T) {
		_, err := LoadObject(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("typed nil pointer", func(t *testing.T) {
		var s *sample
		_, err := LoadObject(s)
		require.Error(t, err)
	})

	t.Run("string delegates to loader", func(t *testing.T) {
		root, err := LoadObject("name: test")
		require.NoError(t, err)
		assert.IsType(t, map[string]interface{}{}, root)
	})

	t.Run("bytes delegates to loader", func(t *testing.T) {
		root, err := LoadObject([]byte(`{"id":1}`))
		require.NoError(t, err)
		assert.IsType(t, map[string]interface{}{}, root)
	})

	t.Run("map returns same reference", func(t *testing.T) {
		obj := map[string]any{"name": "alice"}
		root, err := LoadObject(obj)
		require.NoError(t, err)

		rootMap, ok := root.(map[string]any)
		require.True(t, ok)
		rootMap["role"] = "admin"
		assert.Equal(t, "admin", obj["role"])
	})

	t.Run("pointer returns same reference", func(t *testing.T) {
		obj := &sample{Name: "bob"}
		root, err := LoadObject(obj)
		require.NoError(t, err)
		// Pointers to custom structs are converted to maps for CEL compatibility
		// (CEL cannot evaluate expressions on arbitrary struct types)
		rootMap, ok := root.(map[string]interface{})
		require.True(t, ok, "expected struct pointer to be converted to map")
		assert.Equal(t, "bob", rootMap["Name"])
	})

	t.Run("custom struct converted for CEL compatibility", func(t *testing.T) {
		// This simulates the user's issue: passing a custom struct that needs CEL evaluation
		type ArtifactListItem struct {
			Name    string
			Version string
		}
		obj := &ArtifactListItem{Name: "complex-workflow", Version: "1.0.0"}
		root, err := LoadObject(obj)
		require.NoError(t, err)

		// Verify it's converted to a map that CEL can handle
		rootMap, ok := root.(map[string]interface{})
		require.True(t, ok, "expected custom struct to be converted to map")
		assert.Equal(t, "complex-workflow", rootMap["Name"])
		assert.Equal(t, "1.0.0", rootMap["Version"])
	})

	t.Run("slice containing nil pointer does not panic", func(t *testing.T) {
		var s *sample
		obj := []any{s, &sample{Name: "valid"}}
		root, err := LoadObject(obj)
		require.NoError(t, err)

		result, ok := root.([]interface{})
		require.True(t, ok)
		require.Len(t, result, 2)
		assert.Nil(t, result[0])

		valid, ok := result[1].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "valid", valid["Name"])
	})

	t.Run("nested custom structs converted recursively", func(t *testing.T) {
		type Metadata struct {
			Value string
		}
		type Item struct {
			Name string
			Meta Metadata
		}
		obj := &Item{Name: "test", Meta: Metadata{Value: "data"}}
		root, err := LoadObject(obj)
		require.NoError(t, err)

		rootMap, ok := root.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "test", rootMap["Name"])

		// Nested structs are also converted to maps
		metaVal, ok := rootMap["Meta"].(map[string]interface{})
		require.True(t, ok, "nested struct should also be converted to map")
		assert.Equal(t, "data", metaVal["Value"])
	})
}
