package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSchema_BasicProperties(t *testing.T) {
	schema := `{
		"type": "object",
		"required": ["name", "id"],
		"properties": {
			"name": {
				"type": "string",
				"title": "Full Name",
				"maxLength": 30
			},
			"id": {
				"type": "integer"
			},
			"description": {
				"type": "string",
				"deprecated": true
			},
			"status": {
				"type": "string",
				"enum": ["active", "inactive", "pending"]
			}
		}
	}`

	hints, err := ParseSchema([]byte(schema))
	require.NoError(t, err)
	require.Len(t, hints, 4)

	// name: title→DisplayName, maxLength→MaxWidth, required→Priority+10
	assert.Equal(t, "Full Name", hints["name"].DisplayName)
	assert.Equal(t, 30, hints["name"].MaxWidth)
	assert.True(t, hints["name"].Priority >= 10, "required field should have priority >= 10")

	// id: integer→Align right, required→Priority+10
	assert.Equal(t, "right", hints["id"].Align)
	assert.True(t, hints["id"].Priority >= 10, "required field should have priority >= 10")

	// description: deprecated→Hidden
	assert.True(t, hints["description"].Hidden)

	// status: enum→MaxWidth (longest value "inactive" = 8)
	assert.Equal(t, 8, hints["status"].MaxWidth)
}

func TestParseSchema_NumberType(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"price": {"type": "number"},
			"count": {"type": "integer"}
		}
	}`

	hints, err := ParseSchema([]byte(schema))
	require.NoError(t, err)

	assert.Equal(t, "right", hints["price"].Align)
	assert.Equal(t, "right", hints["count"].Align)
}

func TestParseSchema_FormatWidth(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"created_at": {"type": "string", "format": "date-time"},
			"birthday": {"type": "string", "format": "date"},
			"email": {"type": "string", "format": "email"},
			"website": {"type": "string", "format": "uri"},
			"uid": {"type": "string", "format": "uuid"},
			"ip": {"type": "string", "format": "ipv4"}
		}
	}`

	hints, err := ParseSchema([]byte(schema))
	require.NoError(t, err)

	assert.Equal(t, 26, hints["created_at"].MaxWidth)
	assert.Equal(t, 10, hints["birthday"].MaxWidth)
	assert.Equal(t, 40, hints["email"].MaxWidth)
	assert.Equal(t, 60, hints["website"].MaxWidth)
	assert.Equal(t, 36, hints["uid"].MaxWidth)
	assert.Equal(t, 15, hints["ip"].MaxWidth)
}

func TestParseSchema_PropertyOrder(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"a": {"type": "string"},
			"b": {"type": "string"},
			"c": {"type": "string"}
		}
	}`

	hints, err := ParseSchema([]byte(schema))
	require.NoError(t, err)

	// Properties listed first should have higher priority
	assert.Greater(t, hints["a"].Priority, hints["b"].Priority)
	assert.Greater(t, hints["b"].Priority, hints["c"].Priority)
}

func TestParseSchema_EmptySchema(t *testing.T) {
	hints, err := ParseSchema([]byte(`{}`))
	require.NoError(t, err)
	assert.Empty(t, hints)
}

func TestParseSchema_InvalidJSON(t *testing.T) {
	_, err := ParseSchema([]byte(`not json`))
	assert.Error(t, err)
}

func TestParseSchema_ArrayItems(t *testing.T) {
	// Schema for an array of objects - properties come from "items"
	schema := `{
		"type": "array",
		"items": {
			"type": "object",
			"required": ["id"],
			"properties": {
				"id": {"type": "integer"},
				"label": {"type": "string", "maxLength": 20}
			}
		}
	}`

	hints, err := ParseSchema([]byte(schema))
	require.NoError(t, err)
	require.Len(t, hints, 2)

	assert.Equal(t, "right", hints["id"].Align)
	assert.Equal(t, 20, hints["label"].MaxWidth)
}

func TestParseSchema_MaxLengthWithHeaderFloor(t *testing.T) {
	// When both maxLength and enum are present, maxLength takes precedence.
	// The result is then floored at the header width to prevent truncation.
	schema := `{
		"type": "object",
		"properties": {
			"status": {
				"type": "string",
				"maxLength": 5,
				"enum": ["active", "inactive"]
			}
		}
	}`

	hints, err := ParseSchema([]byte(schema))
	require.NoError(t, err)

	// maxLength=5 would win over enum ("inactive"=8), but header
	// "status" (6 chars) is wider, so MaxWidth floors at 6.
	assert.Equal(t, 6, hints["status"].MaxWidth)
}

func TestParseSchema_MaxWidthFloorsAtHeaderWidth(t *testing.T) {
	// When enum-derived MaxWidth is narrower than the header (title or key),
	// MaxWidth should be bumped to the header width so the header is not truncated.
	schema := `{
		"type": "object",
		"properties": {
			"severity": {
				"type": "string",
				"title": "Severity",
				"enum": ["error", "warn", "info"]
			},
			"short": {
				"type": "string",
				"title": "S",
				"enum": ["longvalue"]
			}
		}
	}`

	hints, err := ParseSchema([]byte(schema))
	require.NoError(t, err)

	// "Severity" (8 chars) > max enum "error" (5 chars) → MaxWidth = 8
	assert.Equal(t, 8, hints["severity"].MaxWidth)
	// "S" (1 char) < max enum "longvalue" (9 chars) → MaxWidth stays 9
	assert.Equal(t, 9, hints["short"].MaxWidth)
}

func TestParseSchema_FlexAutoSet(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"severity": {
				"type": "string",
				"enum": ["error", "warn", "info"]
			},
			"message": {
				"type": "string"
			},
			"count": {
				"type": "integer"
			},
			"timestamp": {
				"type": "string",
				"format": "date-time"
			}
		}
	}`

	hints, err := ParseSchema([]byte(schema))
	require.NoError(t, err)

	// severity has enum → MaxWidth set → not flex
	assert.False(t, hints["severity"].Flex, "enum column should not be flex")
	assert.Greater(t, hints["severity"].MaxWidth, 0)

	// message has no maxLength/enum/format → MaxWidth=0 → flex
	assert.True(t, hints["message"].Flex, "unconstrained string column should be flex")
	assert.Equal(t, 0, hints["message"].MaxWidth)

	// count is integer with no constraints → MaxWidth=0 → flex
	assert.True(t, hints["count"].Flex, "unconstrained integer column should be flex")

	// timestamp has format → MaxWidth set → not flex
	assert.False(t, hints["timestamp"].Flex, "format-constrained column should not be flex")
	assert.Greater(t, hints["timestamp"].MaxWidth, 0)
}

func TestParseSchema_HiddenColumnNotFlex(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"name": { "type": "string" },
			"internal_code": {
				"type": "string",
				"deprecated": true
			}
		}
	}`

	hints, err := ParseSchema([]byte(schema))
	require.NoError(t, err)

	// name: no constraints, not hidden → flex
	assert.True(t, hints["name"].Flex, "visible unconstrained column should be flex")

	// internal_code: deprecated (hidden), no constraints → should NOT be flex
	assert.True(t, hints["internal_code"].Hidden, "deprecated column should be hidden")
	assert.False(t, hints["internal_code"].Flex, "hidden column should not be flex")
}
