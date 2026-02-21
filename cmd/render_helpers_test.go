package cmd

import (
	"testing"

	ui "github.com/oakwood-commons/kvx/internal/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderPlainTextStatus_Basic(t *testing.T) {
	data := map[string]any{
		"title": "Sign in to Entra",
		"url":   "https://microsoft.com/devicelogin",
		"code":  "EH5HFPGJJ",
		"messages": []any{
			"Already authenticated as user@example.com",
			"Use 'myapp auth logout entra' to sign out first",
		},
	}
	ds := &ui.DisplaySchema{
		Version: "v1",
		Status: &ui.StatusDisplayConfig{
			TitleField:   "title",
			MessageField: "messages",
			WaitMessage:  "Waiting for authentication...",
			Actions: []ui.StatusActionConfig{
				{Label: "Copy code", Type: "copy-value", Field: "code"},
				{Label: "Open URL", Type: "open-url", Field: "url"},
			},
			DisplayFields: []ui.StatusFieldDisplay{
				{Label: "URL", Field: "url"},
				{Label: "Code", Field: "code"},
			},
		},
	}

	text, ok := renderPlainTextStatus(data, ds)
	require.True(t, ok)

	assert.Contains(t, text, "Sign in to Entra")
	assert.Contains(t, text, "Already authenticated as user@example.com")
	assert.Contains(t, text, "Use 'myapp auth logout entra' to sign out first")
	assert.Contains(t, text, "URL: ")
	assert.Contains(t, text, "https://microsoft.com/devicelogin")
	assert.Contains(t, text, "Code: EH5HFPGJJ")
	assert.Contains(t, text, "Waiting for authentication...")
}

func TestRenderPlainTextStatus_NilSchema(t *testing.T) {
	text, ok := renderPlainTextStatus(map[string]any{}, nil)
	assert.False(t, ok)
	assert.Empty(t, text)
}

func TestRenderPlainTextStatus_NoStatusConfig(t *testing.T) {
	ds := &ui.DisplaySchema{Version: "v1"}
	text, ok := renderPlainTextStatus(map[string]any{}, ds)
	assert.False(t, ok)
	assert.Empty(t, text)
}

func TestRenderPlainTextStatus_NonMapData(t *testing.T) {
	ds := &ui.DisplaySchema{
		Version: "v1",
		Status:  &ui.StatusDisplayConfig{TitleField: "title"},
	}
	text, ok := renderPlainTextStatus("not a map", ds)
	assert.False(t, ok)
	assert.Empty(t, text)
}

func TestRenderPlainTextStatus_StringMessage(t *testing.T) {
	data := map[string]any{
		"title": "Test",
		"msg":   "single message",
	}
	ds := &ui.DisplaySchema{
		Version: "v1",
		Status: &ui.StatusDisplayConfig{
			TitleField:   "title",
			MessageField: "msg",
		},
	}

	text, ok := renderPlainTextStatus(data, ds)
	require.True(t, ok)
	assert.Contains(t, text, "Test")
	assert.Contains(t, text, "single message")
}

func TestRenderPlainTextStatus_NoWaitMessage(t *testing.T) {
	data := map[string]any{"title": "Test"}
	ds := &ui.DisplaySchema{
		Version: "v1",
		Status:  &ui.StatusDisplayConfig{TitleField: "title"},
	}

	text, ok := renderPlainTextStatus(data, ds)
	require.True(t, ok)
	assert.Contains(t, text, "Test")
	assert.NotContains(t, text, "Waiting")
}

func TestRenderPlainTextStatus_DisplayFieldMissing(t *testing.T) {
	data := map[string]any{"title": "Test"}
	ds := &ui.DisplaySchema{
		Version: "v1",
		Status: &ui.StatusDisplayConfig{
			TitleField: "title",
			DisplayFields: []ui.StatusFieldDisplay{
				{Label: "Value", Field: "missing"},
			},
		},
	}

	text, ok := renderPlainTextStatus(data, ds)
	require.True(t, ok)
	assert.NotContains(t, text, "Value:")
}
