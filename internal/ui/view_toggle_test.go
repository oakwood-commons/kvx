package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// listOfObjects returns a homogeneous array suitable for the list view.
func listOfObjects() []interface{} {
	return []interface{}{
		map[string]interface{}{"name": "alpha", "role": "primary"},
		map[string]interface{}{"name": "beta", "role": "secondary"},
	}
}

func listSchema() *DisplaySchema {
	return &DisplaySchema{
		List: &ListDisplayConfig{
			TitleField:    "name",
			SubtitleField: "role",
		},
		Detail: &DetailDisplayConfig{
			TitleField: "name",
		},
	}
}

func newSchemaModel(t *testing.T, node interface{}, schema *DisplaySchema) *Model {
	t.Helper()
	m := InitialModel(node)
	m.Root = node
	m.Node = node
	m.KeyMode = KeyModeVim
	m.WinWidth = 80
	m.WinHeight = 24
	m.DisplaySchema = schema
	m.updateViewMode(node)
	m.applyLayout(true)
	return &m
}

// ---------------------------------------------------------------------------
// updateViewMode + ShowRawView
// ---------------------------------------------------------------------------

func TestUpdateViewMode_ShowRawViewForcesDefault(t *testing.T) {
	data := listOfObjects()
	m := newSchemaModel(t, data, listSchema())
	require.Equal(t, "list", m.ViewMode)
	require.NotNil(t, m.ListViewState)

	m.ShowRawView = true
	m.updateViewMode(data)

	assert.Equal(t, "", m.ViewMode)
	assert.Nil(t, m.ListViewState)
	assert.Nil(t, m.DetailViewState)
}

func TestUpdateViewMode_ShowRawViewToggleRestoresList(t *testing.T) {
	data := listOfObjects()
	m := newSchemaModel(t, data, listSchema())

	m.ShowRawView = true
	m.updateViewMode(data)
	assert.Equal(t, "", m.ViewMode)

	m.ShowRawView = false
	m.updateViewMode(data)
	assert.Equal(t, "list", m.ViewMode)
	assert.NotNil(t, m.ListViewState)
}

func TestUpdateViewMode_StatusViewOverridesShowRaw(t *testing.T) {
	schema := &DisplaySchema{
		Status: &StatusDisplayConfig{TitleField: "title"},
	}
	m := &Model{
		DisplaySchema: schema,
		ShowRawView:   true,
		WinWidth:      80,
		WinHeight:     24,
	}
	m.updateViewMode(map[string]interface{}{"title": "Signing in"})

	assert.Equal(t, "status", m.ViewMode)
	assert.NotNil(t, m.StatusViewState)
}

// ---------------------------------------------------------------------------
// menuActionToggleView
// ---------------------------------------------------------------------------

func TestMenuActionToggleView_NoOpWithoutSchema(t *testing.T) {
	node := map[string]any{"a": 1}
	m := InitialModel(node)
	m.Root = node
	m.WinWidth = 80
	m.WinHeight = 24
	m.applyLayout(true)

	cmd := menuActionToggleView(&m)

	assert.Nil(t, cmd)
	assert.False(t, m.ShowRawView)
}

func TestMenuActionToggleView_TogglesShowRawView(t *testing.T) {
	m := newSchemaModel(t, listOfObjects(), listSchema())
	require.Equal(t, "list", m.ViewMode)

	menuActionToggleView(m)
	assert.True(t, m.ShowRawView)
	assert.Equal(t, "", m.ViewMode)
	assert.Nil(t, m.ListViewState)

	menuActionToggleView(m)
	assert.False(t, m.ShowRawView)
	assert.Equal(t, "list", m.ViewMode)
	assert.NotNil(t, m.ListViewState)
}

func TestMenuActionToggleView_RestoresDetailViewOnObjectNode(t *testing.T) {
	m := newSchemaModel(t, listOfObjects(), listSchema())
	// Simulate drilling into an item so we are in detail view.
	obj := map[string]interface{}{"name": "alpha", "role": "primary"}
	m.Node = obj
	m.ViewMode = "detail"
	m.DetailViewState = buildDetailViewModel(obj, m.DisplaySchema, m.WinWidth, m.WinHeight)
	m.ListViewState = nil

	menuActionToggleView(m)
	require.True(t, m.ShowRawView)
	require.Equal(t, "", m.ViewMode)

	menuActionToggleView(m)
	assert.False(t, m.ShowRawView)
	assert.Equal(t, "detail", m.ViewMode, "detail view should be restored after toggling back")
	assert.NotNil(t, m.DetailViewState)
}

func TestMenuActionToggleView_BlursInputOnRestore(t *testing.T) {
	m := newSchemaModel(t, listOfObjects(), listSchema())
	m.AllowEditInput = true

	menuActionToggleView(m)
	require.True(t, m.ShowRawView)

	// Simulate the user entering expression mode while in raw view.
	m.InputFocused = true
	m.PathInput.SetValue("_.items")
	m.PathInput.Focus()

	menuActionToggleView(m)
	assert.False(t, m.ShowRawView)
	assert.False(t, m.InputFocused, "toggling back to schema should blur input")
}

// ---------------------------------------------------------------------------
// Key dispatch
// ---------------------------------------------------------------------------

func TestVim_V_MapsToToggleView(t *testing.T) {
	m := testKeyModeModel(KeyModeVim)
	assert.Equal(t, VimActionToggleView, m.handleVimKey("v"))
}

func TestEmacs_AltV_MapsToToggleView(t *testing.T) {
	m := testKeyModeModel(KeyModeEmacs)
	assert.Equal(t, VimActionToggleView, m.handleEmacsKey("alt+v"))
}

// TestVimToggleView_DispatchesToMenuAction exercises the vim helper directly
// so the executeVimAction dispatch path is covered.
func TestVimToggleView_DispatchesToMenuAction(t *testing.T) {
	m := newSchemaModel(t, listOfObjects(), listSchema())
	require.Equal(t, "list", m.ViewMode)

	result, cmd := m.vimToggleView()
	assert.Nil(t, cmd)
	got := result.(*Model)
	assert.True(t, got.ShowRawView, "vimToggleView should flip ShowRawView")
	assert.Equal(t, "", got.ViewMode)
}

// TestExecuteVimAction_ToggleView routes through the vim action dispatcher
// to cover the switch case that maps VimActionToggleView to vimToggleView.
func TestExecuteVimAction_ToggleView(t *testing.T) {
	m := newSchemaModel(t, listOfObjects(), listSchema())
	require.Equal(t, "list", m.ViewMode)

	result, _ := m.executeVimAction(VimActionToggleView)
	got := result.(*Model)
	assert.True(t, got.ShowRawView)
}

// TestExprToggle_BlockedWhenSchemaWithoutRaw covers the guard in
// menuActionExprToggle that suppresses expression mode while a schema view
// is active.
func TestExprToggle_BlockedWhenSchemaWithoutRaw(t *testing.T) {
	m := newSchemaModel(t, listOfObjects(), listSchema())
	m.AllowEditInput = true
	require.False(t, m.ShowRawView)

	cmd := menuActionExprToggle(m)
	assert.Nil(t, cmd)
	assert.False(t, m.InputFocused, "expr toggle must be suppressed while schema view is active")
}

// TestExprToggle_AllowedWhenRawViewActive covers the opposite branch: once the
// user toggles into raw view, expression mode is available again.
func TestExprToggle_AllowedWhenRawViewActive(t *testing.T) {
	m := newSchemaModel(t, listOfObjects(), listSchema())
	m.AllowEditInput = true
	menuActionToggleView(m)
	require.True(t, m.ShowRawView)

	menuActionExprToggle(m)
	assert.True(t, m.InputFocused, "expr toggle should work in raw view")
}

// TestHandleMenuKey_ExprToggle_SchemaGate covers the handleMenuKey guard that
// mirrors the menuActionExprToggle gate for function-key mode.
func TestHandleMenuKey_ExprToggle_SchemaGate(t *testing.T) {
	m := newSchemaModel(t, listOfObjects(), listSchema())
	m.AllowEditInput = true
	m.KeyMode = KeyModeFunction

	// Schema active, raw off -> F6 should be a no-op (guard returns true, nil).
	handled, cmd := m.handleMenuKey("f6")
	assert.True(t, handled, "F6 should be intercepted while schema view is active")
	assert.Nil(t, cmd)

	// Toggle to raw view and confirm F6 now falls through to the real action.
	menuActionToggleView(m)
	require.True(t, m.ShowRawView)
	handled, _ = m.handleMenuKey("f6")
	assert.True(t, handled)
	assert.True(t, m.InputFocused, "F6 in raw view should enter expression mode")
}

func TestListView_V_TogglesRawView(t *testing.T) {
	m := newSchemaModel(t, listOfObjects(), listSchema())
	require.Equal(t, "list", m.ViewMode)

	handled, result, _ := m.handleListViewKey("v")
	assert.True(t, handled)
	got := result.(*Model)
	assert.True(t, got.ShowRawView)
	assert.Equal(t, "", got.ViewMode)
}

func TestDetailView_V_TogglesRawView(t *testing.T) {
	m := newSchemaModel(t, listOfObjects(), listSchema())
	obj := map[string]interface{}{"name": "alpha", "role": "primary"}
	m.Node = obj
	m.ViewMode = "detail"
	m.DetailViewState = buildDetailViewModel(obj, m.DisplaySchema, m.WinWidth, m.WinHeight)

	handled, result, _ := m.handleDetailViewKey("v")
	assert.True(t, handled)
	got := result.(*Model)
	assert.True(t, got.ShowRawView)
	assert.Equal(t, "", got.ViewMode)
}

// ---------------------------------------------------------------------------
// schemaViewApplies + toggle gating
// ---------------------------------------------------------------------------

func TestSchemaViewApplies(t *testing.T) {
	schema := listSchema()
	provider := map[string]interface{}{"name": "alpha", "role": "primary"}
	arr := listOfObjects()

	tests := []struct {
		name           string
		schema         *DisplaySchema
		node           interface{}
		viewMode       string
		showRaw        bool
		preRawViewMode string
		want           bool
	}{
		{"no schema", nil, arr, "", false, "", false},
		{"schema + array node", schema, arr, "list", false, "", true},
		{"schema + array node in raw", schema, arr, "", true, "list", true},
		{"schema + map in detail", schema, provider, "detail", false, "", true},
		{"schema + map in raw from detail", schema, provider, "", true, "detail", true},
		{"schema + scalar in raw", schema, "just a string", "", true, "detail", false},
		{"schema + scalar without context", schema, "just a string", "", false, "", false},
		{"schema + map without context", schema, provider, "", false, "", false},
		{"status view schema is excluded", &DisplaySchema{Status: &StatusDisplayConfig{TitleField: "t"}}, provider, "", false, "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := &Model{
				DisplaySchema:  tc.schema,
				Node:           tc.node,
				ViewMode:       tc.viewMode,
				ShowRawView:    tc.showRaw,
				PreRawViewMode: tc.preRawViewMode,
			}
			assert.Equal(t, tc.want, m.schemaViewApplies())
		})
	}
}

func TestMenuActionToggleView_NoOpAtScalarLeaf(t *testing.T) {
	m := newSchemaModel(t, listOfObjects(), listSchema())

	// Toggle into raw view from list.
	menuActionToggleView(m)
	require.True(t, m.ShowRawView)
	require.Equal(t, "list", m.PreRawViewMode)

	// Drill deep into a scalar leaf so no schema view applies.
	m = m.NavigateTo("alpha", "_.[0].name")

	// v should be a no-op: state preserved.
	menuActionToggleView(m)
	assert.True(t, m.ShowRawView, "raw override must remain when toggle is invalid")
	assert.Equal(t, "list", m.PreRawViewMode, "pre-raw context must be preserved")

	// Navigate back to the parent map.
	result, _ := m.navigateBack()
	m2 := result.(*Model)

	// Now v should successfully restore detail view.
	menuActionToggleView(m2)
	assert.False(t, m2.ShowRawView)
	assert.Equal(t, "detail", m2.ViewMode)
	assert.NotNil(t, m2.DetailViewState)
}

// ---------------------------------------------------------------------------
// Copy behaviour after toggle
// ---------------------------------------------------------------------------

func TestCopy_WorksAfterViewToggle(t *testing.T) {
	captured := withCapturedClipboard(t)
	m := newSchemaModel(t, listOfObjects(), listSchema())
	require.Equal(t, "list", m.ViewMode)

	menuActionToggleView(m)
	require.True(t, m.ShowRawView)
	require.Equal(t, "", m.ViewMode)

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	_ = updated

	assert.NotEmpty(t, *captured, "expected copy to work after toggling to raw view")
}

// ---------------------------------------------------------------------------
// Footer visibility
// ---------------------------------------------------------------------------

// HideCopy controls both `y copy` and `Y copy value` -- see TestFooter_HideCopy_HidesBothBindings.
func TestFooter_ShowViewToggleReflectsSchemaAndRawState(t *testing.T) {
	tests := []struct {
		name           string
		schema         *DisplaySchema
		node           interface{}
		showRaw        bool
		preRaw         string
		viewMode       string
		wantShowToggle bool
		wantHideCopy   bool
	}{
		{"no schema", nil, listOfObjects(), false, "", "", false, false},
		{"schema active on list", listSchema(), listOfObjects(), false, "", "list", true, true},
		{"schema with raw override on array", listSchema(), listOfObjects(), true, "list", "", true, false},
		{"schema with raw at scalar leaf hides toggle", listSchema(), "alpha", true, "detail", "", false, false},
		{"schema at map root without context hides toggle", listSchema(), map[string]interface{}{"a": 1}, false, "", "", false, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := InitialModel(tc.node)
			m.Root = tc.node
			m.Node = tc.node
			m.WinWidth = 80
			m.WinHeight = 24
			m.DisplaySchema = tc.schema
			m.ShowRawView = tc.showRaw
			m.PreRawViewMode = tc.preRaw
			m.ViewMode = tc.viewMode
			m.applyLayout(true)

			state := panelLayoutStateFromModel(&m, PanelLayoutModelOptions{})
			assert.Equal(t, tc.wantShowToggle, state.ShowViewToggle, "ShowViewToggle")
			assert.Equal(t, tc.wantHideCopy, state.HideCopy, "HideCopy")
		})
	}
}

// TestFooter_HideCopy_HidesBothBindings verifies renderFooter suppresses both
// `y copy` and `Y copy value` when hideCopy is true, and shows both when false.
func TestFooter_HideCopy_HidesBothBindings(t *testing.T) {
	tests := []struct {
		name         string
		hideCopy     bool
		wantContains []string
		wantOmits    []string
	}{
		{
			name:         "hideCopy true omits both",
			hideCopy:     true,
			wantContains: []string{"help", "search", "quit"},
			wantOmits:    []string{"copy", "copy value"},
		},
		{
			name:         "hideCopy false shows both",
			hideCopy:     false,
			wantContains: []string{"copy", "copy value"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out := renderFooter(true, true, tc.hideCopy, false, false, 200, KeyModeVim)
			for _, want := range tc.wantContains {
				assert.Contains(t, out, want, "expected footer to contain %q", want)
			}
			for _, omit := range tc.wantOmits {
				assert.NotContains(t, out, omit, "expected footer to omit %q", omit)
			}
		})
	}
}

// TestListView_CopyValue_ConsumedAsNoop verifies pressing Y inside the schema
// list view does not write to the clipboard (the binding is hidden and inert).
func TestListView_CopyValue_ConsumedAsNoop(t *testing.T) {
	captured := withCapturedClipboard(t)
	m := newSchemaModel(t, listOfObjects(), listSchema())
	require.Equal(t, "list", m.ViewMode)

	handled, _, _ := m.handleListViewKey("Y")
	assert.True(t, handled, "list view should consume Y")
	assert.Empty(t, *captured, "Y in list view must not write to clipboard")
}

// TestDetailView_CopyValue_ConsumedAsNoop verifies pressing Y inside the schema
// detail view does not write to the clipboard.
func TestDetailView_CopyValue_ConsumedAsNoop(t *testing.T) {
	captured := withCapturedClipboard(t)
	m := newSchemaModel(t, listOfObjects(), listSchema())
	obj := map[string]interface{}{"name": "alpha", "role": "primary"}
	m.Node = obj
	m.ViewMode = "detail"
	m.DetailViewState = buildDetailViewModel(obj, m.DisplaySchema, m.WinWidth, m.WinHeight)

	handled, _, _ := m.handleDetailViewKey("Y")
	assert.True(t, handled, "detail view should consume Y")
	assert.Empty(t, *captured, "Y in detail view must not write to clipboard")
}
