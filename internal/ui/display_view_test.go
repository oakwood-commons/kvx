package ui

import (
	"strings"
	"testing"

	"github.com/oakwood-commons/kvx/internal/formatter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// buildListViewModel
// ---------------------------------------------------------------------------

func TestBuildListViewModel_Basic(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{"name": "alpha", "desc": "First", "status": "active"},
		map[string]interface{}{"name": "beta", "desc": "Second", "status": "beta"},
	}
	schema := &DisplaySchema{
		List: &ListDisplayConfig{
			TitleField:    "name",
			SubtitleField: "desc",
			BadgeFields:   []string{"status"},
		},
	}

	vm := buildListViewModel(data, schema, 80, 24)
	require.NotNil(t, vm)
	assert.Len(t, vm.Items, 2)
	assert.Equal(t, "alpha", vm.Items[0].Title)
	assert.Equal(t, "First", vm.Items[0].Subtitle)
	assert.Equal(t, []string{"active"}, vm.Items[0].Badges)
	assert.Equal(t, "beta", vm.Items[1].Title)
}

func TestBuildListViewModel_NilSchema(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{"name": "x"},
	}
	assert.Nil(t, buildListViewModel(data, nil, 80, 24))
}

func TestBuildListViewModel_NotArray(t *testing.T) {
	schema := &DisplaySchema{List: &ListDisplayConfig{TitleField: "name"}}
	assert.Nil(t, buildListViewModel("not an array", schema, 80, 24))
}

func TestBuildListViewModel_EmptyArray(t *testing.T) {
	schema := &DisplaySchema{List: &ListDisplayConfig{TitleField: "name"}}
	vm := buildListViewModel([]interface{}{}, schema, 80, 24)
	require.NotNil(t, vm)
	assert.Empty(t, vm.Items)
}

func TestBuildListViewModel_BadgeFromArray(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{
			"name": "svc",
			"tags": []interface{}{"cloud", "aws"},
		},
	}
	schema := &DisplaySchema{
		List: &ListDisplayConfig{
			TitleField:  "name",
			BadgeFields: []string{"tags"},
		},
	}
	vm := buildListViewModel(data, schema, 80, 24)
	require.NotNil(t, vm)
	assert.Contains(t, vm.Items[0].Badges, "cloud")
	assert.Contains(t, vm.Items[0].Badges, "aws")
}

func TestBuildListViewModel_SecondaryFields(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{
			"name":    "alpha",
			"version": "v1.2.3",
			"owner":   "team-x",
		},
	}
	schema := &DisplaySchema{
		List: &ListDisplayConfig{
			TitleField:      "name",
			SecondaryFields: []string{"version", "owner"},
		},
	}
	vm := buildListViewModel(data, schema, 80, 24)
	require.NotNil(t, vm)
	assert.Contains(t, vm.Items[0].Secondary, "v1.2.3")
	assert.Contains(t, vm.Items[0].Secondary, "team-x")
}

// ---------------------------------------------------------------------------
// renderListView
// ---------------------------------------------------------------------------

func TestRenderListView_ContainsTitles(t *testing.T) {
	data := []interface{}{
		map[string]interface{}{"name": "alpha", "desc": "First provider"},
		map[string]interface{}{"name": "beta", "desc": "Second provider"},
	}
	schema := &DisplaySchema{
		CollectionTitle: "Providers",
		List: &ListDisplayConfig{
			TitleField:    "name",
			SubtitleField: "desc",
		},
	}
	vm := buildListViewModel(data, schema, 60, 20)
	require.NotNil(t, vm)
	content := renderListView(vm, schema, false)
	assert.Contains(t, content, "alpha")
	assert.Contains(t, content, "beta")
}

func TestRenderListView_EmptyList(t *testing.T) {
	schema := &DisplaySchema{
		List: &ListDisplayConfig{TitleField: "name"},
	}
	vm := buildListViewModel([]interface{}{}, schema, 60, 20)
	require.NotNil(t, vm)
	content := renderListView(vm, schema, false)
	assert.Contains(t, content, "empty")
}

// ---------------------------------------------------------------------------
// filterListItems
// ---------------------------------------------------------------------------

func TestFilterListItems(t *testing.T) {
	items := []ListViewItem{
		{Index: 0, Title: "aws-ec2", Subtitle: "EC2 instances"},
		{Index: 1, Title: "gcp-gke", Subtitle: "Kubernetes"},
		{Index: 2, Title: "aws-rds", Subtitle: "Databases"},
	}

	lv := &ListViewModel{Items: items, Filter: "aws"}
	filtered := filterListItems(lv)
	assert.Len(t, filtered, 2)
	assert.Equal(t, "aws-ec2", filtered[0].Title)
	assert.Equal(t, "aws-rds", filtered[1].Title)

	lv.Filter = "kubernetes"
	filtered2 := filterListItems(lv)
	assert.Len(t, filtered2, 1)
	assert.Equal(t, "gcp-gke", filtered2[0].Title)

	lv.Filter = ""
	all := filterListItems(lv)
	assert.Len(t, all, 3)
}

// ---------------------------------------------------------------------------
// isHomogeneousObjectArray
// ---------------------------------------------------------------------------

func TestIsHomogeneousObjectArray(t *testing.T) {
	tests := []struct {
		name   string
		input  interface{}
		expect bool
	}{
		{"array of objects", []interface{}{
			map[string]interface{}{"a": 1},
			map[string]interface{}{"b": 2},
		}, true},
		{"empty array", []interface{}{}, false},
		{"not array", "hello", false},
		{"array of strings", []interface{}{"a", "b"}, false},
		{"mixed array", []interface{}{
			map[string]interface{}{"a": 1},
			"string",
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, isHomogeneousObjectArray(tt.input))
		})
	}
}

// ---------------------------------------------------------------------------
// collectObjectKeys
// ---------------------------------------------------------------------------

func TestCollectObjectKeys(t *testing.T) {
	obj := map[string]interface{}{"name": "a", "version": "v1", "secret": "x"}
	keys := collectObjectKeys(obj, []string{"secret"})
	assert.Contains(t, keys, "name")
	assert.Contains(t, keys, "version")
	assert.NotContains(t, keys, "secret")
}

// ---------------------------------------------------------------------------
// wrapAtWidth
// ---------------------------------------------------------------------------

func TestWrapAtWidth(t *testing.T) {
	// Short string stays on one line
	result := wrapAtWidth("hello", 80)
	assert.Equal(t, "hello", result)

	// Long text wraps on word boundaries
	long := "the quick brown fox jumps over the lazy dog and keeps on running"
	wrapped := wrapAtWidth(long, 30)
	lines := strings.Split(wrapped, "\n")
	assert.Greater(t, len(lines), 1)
}

// ---------------------------------------------------------------------------
// stringify
// ---------------------------------------------------------------------------

func TestStringify(t *testing.T) {
	assert.Equal(t, "hello", formatter.Stringify("hello"))
	assert.Equal(t, "42", formatter.Stringify(42))
	assert.Equal(t, "3.14", formatter.Stringify(3.14))
	assert.Equal(t, "true", formatter.Stringify(true))
	assert.Equal(t, "", formatter.Stringify(nil))
}

// ---------------------------------------------------------------------------
// buildDetailViewModel
// ---------------------------------------------------------------------------

func TestBuildDetailViewModel_Basic(t *testing.T) {
	obj := map[string]interface{}{
		"name":    "aws-ec2",
		"version": "v3.12",
		"desc":    "EC2 instances",
		"tags":    []interface{}{"cloud", "aws"},
	}
	schema := &DisplaySchema{
		Detail: &DetailDisplayConfig{
			TitleField: "name",
			Sections: []DetailSection{
				{Title: "", Fields: []string{"version"}, Layout: "inline"},
				{Title: "Description", Fields: []string{"desc"}, Layout: "paragraph"},
				{Title: "Tags", Fields: []string{"tags"}, Layout: "tags"},
			},
		},
	}

	dv := buildDetailViewModel(obj, schema, 60, 20)
	require.NotNil(t, dv)
	assert.Equal(t, "aws-ec2", dv.TitleText)
	assert.Len(t, dv.Sections, 3)
}

func TestBuildDetailViewModel_NilSchema(t *testing.T) {
	obj := map[string]interface{}{"name": "x"}
	assert.Nil(t, buildDetailViewModel(obj, nil, 60, 20))
}

func TestBuildDetailViewModel_NotObject(t *testing.T) {
	schema := &DisplaySchema{Detail: &DetailDisplayConfig{TitleField: "name"}}
	assert.Nil(t, buildDetailViewModel("not an object", schema, 60, 20))
}

func TestBuildDetailViewModel_HiddenFields(t *testing.T) {
	obj := map[string]interface{}{
		"name":   "test",
		"secret": "hidden-value",
		"public": "visible",
	}
	schema := &DisplaySchema{
		Detail: &DetailDisplayConfig{
			TitleField:   "name",
			HiddenFields: []string{"secret"},
			Sections: []DetailSection{
				{Fields: []string{"public", "secret"}, Layout: "table"},
			},
		},
	}

	dv := buildDetailViewModel(obj, schema, 60, 20)
	require.NotNil(t, dv)
	// The rendered content should not contain "hidden-value"
	content := renderDetailView(dv, schema, false)
	assert.NotContains(t, content, "hidden-value")
}

// ---------------------------------------------------------------------------
// renderDetailView
// ---------------------------------------------------------------------------

func TestRenderDetailView_ContainsSections(t *testing.T) {
	obj := map[string]interface{}{
		"name":    "test-svc",
		"version": "v1.0",
		"desc":    "A test service for unit tests",
	}
	schema := &DisplaySchema{
		Detail: &DetailDisplayConfig{
			TitleField: "name",
			Sections: []DetailSection{
				{Title: "Info", Fields: []string{"version"}, Layout: "inline"},
				{Title: "About", Fields: []string{"desc"}, Layout: "paragraph"},
			},
		},
	}

	dv := buildDetailViewModel(obj, schema, 60, 20)
	require.NotNil(t, dv)
	content := renderDetailView(dv, schema, false)
	// Title is rendered in the panel border, not in the content area.
	assert.NotContains(t, content, "test-svc")
	assert.Contains(t, content, "v1.0")
	assert.Contains(t, content, "A test service for unit tests")
}

// ---------------------------------------------------------------------------
// updateViewMode
// ---------------------------------------------------------------------------

func TestUpdateViewMode_ListMode(t *testing.T) {
	m := &Model{
		DisplaySchema: &DisplaySchema{
			List: &ListDisplayConfig{TitleField: "name"},
		},
		WinWidth:  80,
		WinHeight: 24,
	}
	data := []interface{}{
		map[string]interface{}{"name": "a"},
		map[string]interface{}{"name": "b"},
	}
	m.updateViewMode(data)
	assert.Equal(t, "list", m.ViewMode)
	assert.NotNil(t, m.ListViewState)
}

func TestUpdateViewMode_DefaultWhenNoSchema(t *testing.T) {
	m := &Model{}
	m.updateViewMode([]interface{}{map[string]interface{}{"a": 1}})
	assert.Equal(t, "", m.ViewMode)
	assert.Nil(t, m.ListViewState)
}

func TestUpdateViewMode_DefaultForScalar(t *testing.T) {
	m := &Model{
		DisplaySchema: &DisplaySchema{
			List: &ListDisplayConfig{TitleField: "name"},
		},
	}
	m.updateViewMode("just a string")
	assert.Equal(t, "", m.ViewMode)
}

func TestUpdateViewMode_DetailAfterList(t *testing.T) {
	m := &Model{
		DisplaySchema: &DisplaySchema{
			List:   &ListDisplayConfig{TitleField: "name"},
			Detail: &DetailDisplayConfig{TitleField: "name"},
		},
		ViewMode:  "list",
		WinWidth:  80,
		WinHeight: 24,
	}
	obj := map[string]interface{}{"name": "drilled-in"}
	m.updateViewMode(obj)
	assert.Equal(t, "detail", m.ViewMode)
	assert.NotNil(t, m.DetailViewState)
}

// ---------------------------------------------------------------------------
// resolveViewAction — keymap mode tests
// ---------------------------------------------------------------------------

func TestResolveViewAction_ArrowKeysAllModes(t *testing.T) {
	// Arrow keys must work in every mode.
	for _, mode := range []KeyMode{KeyModeVim, KeyModeEmacs, KeyModeFunction} {
		m := &Model{KeyMode: mode}
		assert.Equal(t, VimActionUp, m.resolveViewAction("up"), "mode=%s up", mode)
		assert.Equal(t, VimActionDown, m.resolveViewAction("down"), "mode=%s down", mode)
		assert.Equal(t, VimActionBack, m.resolveViewAction("left"), "mode=%s left", mode)
		assert.Equal(t, VimActionForward, m.resolveViewAction("right"), "mode=%s right", mode)
		assert.Equal(t, VimActionEnter, m.resolveViewAction("enter"), "mode=%s enter", mode)
		assert.Equal(t, VimActionQuit, m.resolveViewAction("ctrl+c"), "mode=%s ctrl+c", mode)
		assert.Equal(t, VimActionHelp, m.resolveViewAction("f1"), "mode=%s f1", mode)
		assert.Equal(t, VimActionTop, m.resolveViewAction("home"), "mode=%s home", mode)
		assert.Equal(t, VimActionBottom, m.resolveViewAction("end"), "mode=%s end", mode)
	}
}

func TestResolveViewAction_VimLetterKeys(t *testing.T) {
	m := &Model{KeyMode: KeyModeVim}
	assert.Equal(t, VimActionUp, m.resolveViewAction("k"))
	assert.Equal(t, VimActionDown, m.resolveViewAction("j"))
	assert.Equal(t, VimActionBack, m.resolveViewAction("h"))
	assert.Equal(t, VimActionForward, m.resolveViewAction("l"))
	assert.Equal(t, VimActionTop, m.resolveViewAction("g"))    // pending-g → top
	assert.Equal(t, VimActionBottom, m.resolveViewAction("G")) //nolint:goconst
	assert.Equal(t, VimActionHelp, m.resolveViewAction("?"))
	assert.Equal(t, VimActionQuit, m.resolveViewAction("q"))
	assert.Equal(t, VimActionSearch, m.resolveViewAction("/"))
}

func TestResolveViewAction_EmacsKeys(t *testing.T) {
	m := &Model{KeyMode: KeyModeEmacs}
	assert.Equal(t, VimActionUp, m.resolveViewAction("ctrl+p"))
	assert.Equal(t, VimActionDown, m.resolveViewAction("ctrl+n"))
	assert.Equal(t, VimActionBack, m.resolveViewAction("ctrl+b"))
	assert.Equal(t, VimActionForward, m.resolveViewAction("ctrl+f"))
	assert.Equal(t, VimActionQuit, m.resolveViewAction("ctrl+q"))
	// Vim letter keys should NOT resolve in emacs mode.
	assert.Equal(t, VimActionNone, m.resolveViewAction("j"))
	assert.Equal(t, VimActionNone, m.resolveViewAction("k"))
	assert.Equal(t, VimActionNone, m.resolveViewAction("h"))
	assert.Equal(t, VimActionNone, m.resolveViewAction("l"))
}

func TestResolveViewAction_FunctionMode_NoLetterKeys(t *testing.T) {
	m := &Model{KeyMode: KeyModeFunction}
	// Single-letter keys must NOT resolve in function mode.
	for _, key := range []string{"j", "k", "h", "l", "g", "G", "q", "?", "/"} {
		assert.Equal(t, VimActionNone, m.resolveViewAction(key), "function mode: %q should be none", key)
	}
	// Ctrl+ keys from emacs should also be none.
	for _, key := range []string{"ctrl+n", "ctrl+p", "ctrl+b", "ctrl+f"} {
		assert.Equal(t, VimActionNone, m.resolveViewAction(key), "function mode: %q should be none", key)
	}
}

// ---------------------------------------------------------------------------
// handleListViewKey — keymap mode integration
// ---------------------------------------------------------------------------

func listModel(mode KeyMode) *Model {
	data := []interface{}{
		map[string]interface{}{"name": "alpha"},
		map[string]interface{}{"name": "beta"},
		map[string]interface{}{"name": "gamma"},
	}
	schema := &DisplaySchema{List: &ListDisplayConfig{TitleField: "name"}}
	return &Model{
		KeyMode:       mode,
		DisplaySchema: schema,
		ViewMode:      "list",
		Node:          data,
		Path:          "_",
		ListViewState: buildListViewModel(data, schema, 80, 24),
		WinWidth:      80,
		WinHeight:     24,
	}
}

func TestListViewKey_VimNav(t *testing.T) {
	m := listModel(KeyModeVim)
	// j moves down
	handled, _, _ := m.handleListViewKey("j")
	assert.True(t, handled)
	assert.Equal(t, 1, m.ListViewState.Selected)
	// k moves up
	handled, _, _ = m.handleListViewKey("k")
	assert.True(t, handled)
	assert.Equal(t, 0, m.ListViewState.Selected)
}

func TestListViewKey_EmacsNav(t *testing.T) {
	m := listModel(KeyModeEmacs)
	// ctrl+n moves down
	handled, _, _ := m.handleListViewKey("ctrl+n")
	assert.True(t, handled)
	assert.Equal(t, 1, m.ListViewState.Selected)
	// ctrl+p moves up
	handled, _, _ = m.handleListViewKey("ctrl+p")
	assert.True(t, handled)
	assert.Equal(t, 0, m.ListViewState.Selected)
	// vim "j" should NOT move — it becomes a filter character
	m.ListViewState.Selected = 0
	handled, _, _ = m.handleListViewKey("j")
	assert.True(t, handled, "j should be handled as type-ahead filter in emacs mode")
	assert.Equal(t, "j", m.ListViewState.Filter)
}

func TestListViewKey_FunctionMode_LettersFilter(t *testing.T) {
	m := listModel(KeyModeFunction)
	// In function mode, "j" should be a type-ahead filter character
	handled, _, _ := m.handleListViewKey("j")
	assert.True(t, handled)
	assert.Equal(t, "j", m.ListViewState.Filter)
	assert.Equal(t, 0, m.ListViewState.Selected)
	// Arrow keys still work
	m.ListViewState.Filter = ""
	handled, _, _ = m.handleListViewKey("down")
	assert.True(t, handled)
	assert.Equal(t, 1, m.ListViewState.Selected)
}
