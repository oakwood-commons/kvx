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

func TestParseSchema_MaxLengthOverridesEnum(t *testing.T) {
	// When both maxLength and enum are present, maxLength takes precedence
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

	// maxLength=5 should win over enum longest ("inactive"=8)
	assert.Equal(t, 5, hints["status"].MaxWidth)
}
