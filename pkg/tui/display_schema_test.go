package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// ParseDisplaySchema (standalone document)
// ---------------------------------------------------------------------------

func TestParseDisplaySchema_Full(t *testing.T) {
	doc := `{
		"displaySchema": "v1",
		"icon": "📦",
		"collectionTitle": "Providers",
		"list": {
			"titleField": "name",
			"subtitleField": "description",
			"subtitleMaxLines": 2,
			"badgeFields": ["status", "type"],
			"secondaryFields": ["version", "maintainer"],
			"arrayStyle": "bullet"
		},
		"detail": {
			"titleField": "name",
			"hiddenFields": ["internal_id"],
			"sections": [
				{
					"title": "",
					"fields": ["version", "type", "status"],
					"layout": "inline"
				},
				{
					"title": "Description",
					"fields": ["description"],
					"layout": "paragraph"
				},
				{
					"title": "Tags",
					"fields": ["tags"],
					"layout": "tags"
				},
				{
					"title": "Details",
					"fields": ["maintainer", "resources"],
					"layout": "table"
				}
			]
		}
	}`

	ds, err := ParseDisplaySchema([]byte(doc))
	require.NoError(t, err)
	require.NotNil(t, ds)

	assert.Equal(t, "v1", ds.Version)
	assert.Equal(t, "📦", ds.Icon)
	assert.Equal(t, "Providers", ds.CollectionTitle)

	// List config
	require.NotNil(t, ds.List)
	assert.Equal(t, "name", ds.List.TitleField)
	assert.Equal(t, "description", ds.List.SubtitleField)
	assert.Equal(t, 2, ds.List.SubtitleMaxLines)
	assert.Equal(t, []string{"status", "type"}, ds.List.BadgeFields)
	assert.Equal(t, []string{"version", "maintainer"}, ds.List.SecondaryFields)
	assert.Equal(t, "bullet", ds.List.ArrayStyle)

	// Detail config
	require.NotNil(t, ds.Detail)
	assert.Equal(t, "name", ds.Detail.TitleField)
	assert.Equal(t, []string{"internal_id"}, ds.Detail.HiddenFields)
	require.Len(t, ds.Detail.Sections, 4)
	assert.Equal(t, "inline", ds.Detail.Sections[0].Layout)
	assert.Equal(t, "paragraph", ds.Detail.Sections[1].Layout)
	assert.Equal(t, "tags", ds.Detail.Sections[2].Layout)
	assert.Equal(t, "table", ds.Detail.Sections[3].Layout)
}

func TestParseDisplaySchema_Minimal(t *testing.T) {
	doc := `{
		"displaySchema": "v1",
		"list": {
			"titleField": "name"
		}
	}`

	ds, err := ParseDisplaySchema([]byte(doc))
	require.NoError(t, err)
	require.NotNil(t, ds)
	assert.Equal(t, "name", ds.List.TitleField)
	assert.Nil(t, ds.Detail)
	assert.Empty(t, ds.Icon)
}

func TestParseDisplaySchema_MissingKey(t *testing.T) {
	_, err := ParseDisplaySchema([]byte(`{"icon": "📦"}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "displaySchema")
}

func TestParseDisplaySchema_InvalidJSON(t *testing.T) {
	_, err := ParseDisplaySchema([]byte(`not json`))
	assert.Error(t, err)
}

func TestParseDisplaySchema_ListMissingTitleField(t *testing.T) {
	doc := `{
		"displaySchema": "v1",
		"list": {
			"subtitleField": "desc"
		}
	}`
	_, err := ParseDisplaySchema([]byte(doc))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "titleField")
}

func TestParseDisplaySchema_InvalidLayout(t *testing.T) {
	doc := `{
		"displaySchema": "v1",
		"detail": {
			"titleField": "name",
			"sections": [
				{"fields": ["a"], "layout": "fancy"}
			]
		}
	}`
	_, err := ParseDisplaySchema([]byte(doc))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fancy")
}

func TestParseDisplaySchema_EmptySectionFields(t *testing.T) {
	doc := `{
		"displaySchema": "v1",
		"detail": {
			"titleField": "name",
			"sections": [
				{"fields": [], "layout": "table"}
			]
		}
	}`
	_, err := ParseDisplaySchema([]byte(doc))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fields")
}

// ---------------------------------------------------------------------------
// ParseSchemaWithDisplay (JSON Schema with x-kvx-* extensions)
// ---------------------------------------------------------------------------

func TestParseSchemaWithDisplay_XKvxExtensions(t *testing.T) {
	schema := `{
		"type": "array",
		"x-kvx-icon": "🔧",
		"x-kvx-collectionTitle": "Services",
		"x-kvx-list": {
			"titleField": "name",
			"subtitleField": "desc",
			"badgeFields": ["status"]
		},
		"x-kvx-detail": {
			"titleField": "name",
			"sections": [
				{"fields": ["desc"], "layout": "paragraph"},
				{"fields": ["version", "status"], "layout": "inline"}
			]
		},
		"items": {
			"type": "object",
			"required": ["name"],
			"properties": {
				"name": {"type": "string", "title": "Service Name"},
				"desc": {"type": "string"},
				"version": {"type": "string"},
				"status": {"type": "string", "enum": ["active", "inactive"]}
			}
		}
	}`

	hints, ds, err := ParseSchemaWithDisplay([]byte(schema))
	require.NoError(t, err)

	// Column hints still work
	require.NotEmpty(t, hints)
	assert.Equal(t, "Service Name", hints["name"].DisplayName)
	assert.Equal(t, 8, hints["status"].MaxWidth) // "inactive" = 8

	// Display schema extracted
	require.NotNil(t, ds)
	assert.Equal(t, "🔧", ds.Icon)
	assert.Equal(t, "Services", ds.CollectionTitle)
	require.NotNil(t, ds.List)
	assert.Equal(t, "name", ds.List.TitleField)
	assert.Equal(t, "desc", ds.List.SubtitleField)
	assert.Equal(t, []string{"status"}, ds.List.BadgeFields)
	require.NotNil(t, ds.Detail)
	require.Len(t, ds.Detail.Sections, 2)
}

func TestParseSchemaWithDisplay_NoExtensions(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"id": {"type": "integer"},
			"name": {"type": "string"}
		}
	}`

	hints, ds, err := ParseSchemaWithDisplay([]byte(schema))
	require.NoError(t, err)
	require.NotEmpty(t, hints)
	assert.Nil(t, ds, "no x-kvx-* extensions → nil display schema")
}

func TestParseSchemaWithDisplay_BackwardCompatible(t *testing.T) {
	// ParseSchema should still work and return same hints
	schema := `{
		"type": "object",
		"required": ["id"],
		"properties": {
			"id": {"type": "integer"},
			"label": {"type": "string", "maxLength": 20}
		}
	}`

	hintsOld, err := ParseSchema([]byte(schema))
	require.NoError(t, err)

	hintsNew, ds, err := ParseSchemaWithDisplay([]byte(schema))
	require.NoError(t, err)
	assert.Nil(t, ds)
	assert.Equal(t, hintsOld, hintsNew)
}

func TestParseSchemaWithDisplay_ObjectLevelExtensions(t *testing.T) {
	// x-kvx-* on a type=object schema (not array)
	schema := `{
		"type": "object",
		"x-kvx-icon": "👤",
		"x-kvx-collectionTitle": "User",
		"x-kvx-detail": {
			"titleField": "fullName",
			"sections": [
				{"fields": ["email", "phone"], "layout": "table"}
			]
		},
		"properties": {
			"fullName": {"type": "string"},
			"email": {"type": "string"},
			"phone": {"type": "string"}
		}
	}`

	_, ds, err := ParseSchemaWithDisplay([]byte(schema))
	require.NoError(t, err)
	require.NotNil(t, ds)
	assert.Equal(t, "👤", ds.Icon)
	assert.Equal(t, "User", ds.CollectionTitle)
	require.NotNil(t, ds.Detail)
	assert.Equal(t, "fullName", ds.Detail.TitleField)
}

// ---------------------------------------------------------------------------
// ParseDisplaySchema – x-kvx-status (standalone)
// ---------------------------------------------------------------------------

func TestParseDisplaySchema_Status(t *testing.T) {
	doc := `{
		"displaySchema": "v1",
		"x-kvx-status": {
			"titleField": "title",
			"messageField": "messages",
			"waitMessage": "Waiting...",
			"successMessage": "Done!",
			"timeout": "30s",
			"doneBehavior": "wait-for-key",
			"doneDelay": "3s",
			"displayFields": [
				{"label": "URL", "field": "url"},
				{"label": "Code", "field": "code"}
			],
			"actions": [
				{
					"label": "Copy code",
					"type": "copy-value",
					"field": "code",
					"keys": {"vim": "c", "emacs": "alt+c", "function": "f2"}
				},
				{
					"label": "Open URL",
					"type": "open-url",
					"field": "url",
					"keys": {"vim": "o", "emacs": "alt+o", "function": "f3"}
				}
			]
		}
	}`

	ds, err := ParseDisplaySchema([]byte(doc))
	require.NoError(t, err)
	require.NotNil(t, ds)
	require.NotNil(t, ds.Status)

	assert.Equal(t, "title", ds.Status.TitleField)
	assert.Equal(t, "messages", ds.Status.MessageField)
	assert.Equal(t, "Waiting...", ds.Status.WaitMessage)
	assert.Equal(t, "Done!", ds.Status.SuccessMessage)
	assert.Equal(t, "30s", ds.Status.Timeout)
	assert.Equal(t, DoneBehaviorWaitForKey, ds.Status.DoneBehavior)
	assert.Equal(t, "3s", ds.Status.DoneDelay)

	require.Len(t, ds.Status.DisplayFields, 2)
	assert.Equal(t, "URL", ds.Status.DisplayFields[0].Label)
	assert.Equal(t, "url", ds.Status.DisplayFields[0].Field)
	assert.Equal(t, "Code", ds.Status.DisplayFields[1].Label)
	assert.Equal(t, "code", ds.Status.DisplayFields[1].Field)

	require.Len(t, ds.Status.Actions, 2)

	assert.Equal(t, "Copy code", ds.Status.Actions[0].Label)
	assert.Equal(t, "copy-value", ds.Status.Actions[0].Type)
	assert.Equal(t, "code", ds.Status.Actions[0].Field)
	assert.Equal(t, "c", ds.Status.Actions[0].Keys.Vim)
	assert.Equal(t, "alt+c", ds.Status.Actions[0].Keys.Emacs)
	assert.Equal(t, "f2", ds.Status.Actions[0].Keys.Function)

	assert.Equal(t, "Open URL", ds.Status.Actions[1].Label)
	assert.Equal(t, "open-url", ds.Status.Actions[1].Type)
	assert.Equal(t, "url", ds.Status.Actions[1].Field)
}

func TestParseDisplaySchema_StatusMissingTitleField(t *testing.T) {
	doc := `{
		"displaySchema": "v1",
		"x-kvx-status": {
			"messageField": "messages"
		}
	}`
	_, err := ParseDisplaySchema([]byte(doc))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "titleField")
}

func TestParseDisplaySchema_StatusInvalidActionType(t *testing.T) {
	doc := `{
		"displaySchema": "v1",
		"x-kvx-status": {
			"titleField": "title",
			"actions": [
				{"label": "Do thing", "type": "unknown-action", "field": "x"}
			]
		}
	}`
	_, err := ParseDisplaySchema([]byte(doc))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown-action")
}

func TestParseDisplaySchema_StatusActionMissingLabel(t *testing.T) {
	doc := `{
		"displaySchema": "v1",
		"x-kvx-status": {
			"titleField": "title",
			"actions": [
				{"type": "copy-value", "field": "code"}
			]
		}
	}`
	_, err := ParseDisplaySchema([]byte(doc))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "label")
}

func TestParseDisplaySchema_StatusMinimal(t *testing.T) {
	doc := `{
		"displaySchema": "v1",
		"x-kvx-status": {
			"titleField": "title"
		}
	}`
	ds, err := ParseDisplaySchema([]byte(doc))
	require.NoError(t, err)
	require.NotNil(t, ds.Status)
	assert.Equal(t, "title", ds.Status.TitleField)
	assert.Empty(t, ds.Status.Actions)
}

// ---------------------------------------------------------------------------
// ParseSchemaWithDisplay – x-kvx-status in JSON Schema
// ---------------------------------------------------------------------------

func TestParseSchemaWithDisplay_StatusExtension(t *testing.T) {
	schema := `{
		"type": "object",
		"x-kvx-status": {
			"titleField": "title",
			"messageField": "msg",
			"waitMessage": "Please wait...",
			"actions": [
				{
					"label": "Copy token",
					"type": "copy-value",
					"field": "token",
					"keys": {"vim": "c", "emacs": "alt+c", "function": "f2"}
				}
			]
		},
		"properties": {
			"title": {"type": "string"},
			"msg": {"type": "string"},
			"token": {"type": "string"}
		}
	}`

	_, ds, err := ParseSchemaWithDisplay([]byte(schema))
	require.NoError(t, err)
	require.NotNil(t, ds)
	require.NotNil(t, ds.Status)
	assert.Equal(t, "title", ds.Status.TitleField)
	assert.Equal(t, "msg", ds.Status.MessageField)
	assert.Equal(t, "Please wait...", ds.Status.WaitMessage)
	require.Len(t, ds.Status.Actions, 1)
	assert.Equal(t, "Copy token", ds.Status.Actions[0].Label)
}

// ---------------------------------------------------------------------------
// Layout constants
// ---------------------------------------------------------------------------

func TestDisplaySchemaLayoutConstants(t *testing.T) {
	assert.Equal(t, "inline", DisplaySchemaLayoutInline)
	assert.Equal(t, "paragraph", DisplaySchemaLayoutParagraph)
	assert.Equal(t, "tags", DisplaySchemaLayoutTags)
	assert.Equal(t, "table", DisplaySchemaLayoutTable)
}

// ---------------------------------------------------------------------------
// DoneBehavior constants
// ---------------------------------------------------------------------------

func TestDoneBehaviorConstants(t *testing.T) {
	assert.Equal(t, "exit-after-delay", DoneBehaviorExitAfterDelay)
	assert.Equal(t, "wait-for-key", DoneBehaviorWaitForKey)
}
