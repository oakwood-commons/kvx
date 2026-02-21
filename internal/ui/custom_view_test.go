package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Compile-time interface conformance
// ---------------------------------------------------------------------------

var (
	_ CustomView = (*ListViewModel)(nil)
	_ CustomView = (*DetailViewModel)(nil)
	_ CustomView = (*StatusViewModel)(nil)
)

// ---------------------------------------------------------------------------
// activeCustomView
// ---------------------------------------------------------------------------

func TestActiveCustomView_List(t *testing.T) {
	m := &Model{
		ViewMode:      "list",
		ListViewState: &ListViewModel{CollectionTitle: "Providers"},
	}
	cv := m.activeCustomView()
	require.NotNil(t, cv)
	assert.Equal(t, "Providers", cv.Title())
}

func TestActiveCustomView_Detail(t *testing.T) {
	m := &Model{
		ViewMode:        "detail",
		DetailViewState: &DetailViewModel{TitleText: "aws-ec2"},
	}
	cv := m.activeCustomView()
	require.NotNil(t, cv)
	assert.Equal(t, "aws-ec2", cv.Title())
}

func TestActiveCustomView_Status(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)
	m := &Model{
		ViewMode:        "status",
		StatusViewState: sv,
	}
	cv := m.activeCustomView()
	require.NotNil(t, cv)
	assert.Equal(t, "Sign in to Entra", cv.Title())
}

func TestActiveCustomView_DefaultTable(t *testing.T) {
	m := &Model{ViewMode: ""}
	assert.Nil(t, m.activeCustomView())
}

func TestActiveCustomView_NilState(t *testing.T) {
	// ViewMode is set but the corresponding state is nil.
	for _, mode := range []string{"list", "detail", "status"} {
		m := &Model{ViewMode: mode}
		assert.Nil(t, m.activeCustomView(), "mode=%s with nil state", mode)
	}
}

// ---------------------------------------------------------------------------
// ListViewModel — CustomView methods
// ---------------------------------------------------------------------------

func TestListViewModel_Title(t *testing.T) {
	lv := &ListViewModel{CollectionTitle: "📦 Providers"}
	assert.Equal(t, "📦 Providers", lv.Title())
}

func TestListViewModel_Title_Empty(t *testing.T) {
	lv := &ListViewModel{}
	assert.Equal(t, "", lv.Title())
}

func TestListViewModel_FooterBar(t *testing.T) {
	lv := &ListViewModel{}
	assert.Equal(t, "", lv.FooterBar())
}

func TestListViewModel_HandlesSearch(t *testing.T) {
	lv := &ListViewModel{}
	assert.True(t, lv.HandlesSearch())
}

func TestListViewModel_Init(t *testing.T) {
	lv := &ListViewModel{}
	assert.Nil(t, lv.Init())
}

func TestListViewModel_SearchTitle(t *testing.T) {
	lv := &ListViewModel{}
	assert.Equal(t, "", lv.SearchTitle())
}

func TestListViewModel_FlashMessage(t *testing.T) {
	lv := &ListViewModel{}
	msg, isErr := lv.FlashMessage()
	assert.Equal(t, "", msg)
	assert.False(t, isErr)
}

func TestListViewModel_RowCount_Basic(t *testing.T) {
	schema := &DisplaySchema{
		List: &ListDisplayConfig{TitleField: "name"},
	}
	data := []interface{}{
		map[string]interface{}{"name": "a"},
		map[string]interface{}{"name": "b"},
		map[string]interface{}{"name": "c"},
	}
	lv := buildListViewModel(data, schema, 80, 24)
	require.NotNil(t, lv)

	count, sel, label := lv.RowCount()
	assert.Equal(t, 3, count)
	assert.Equal(t, 1, sel)
	assert.Equal(t, "list: 1/3", label)
}

func TestListViewModel_RowCount_WithSearchSuffix(t *testing.T) {
	lv := &ListViewModel{
		Items: []ListViewItem{
			{Title: "alpha"},
			{Title: "beta"},
		},
		Selected:    1,
		SearchQuery: "al",
	}
	_, _, label := lv.RowCount()
	assert.Contains(t, label, "/al")
}

func TestListViewModel_RowCount_WithFilterSuffix(t *testing.T) {
	lv := &ListViewModel{
		Items: []ListViewItem{
			{Title: "alpha"},
			{Title: "beta"},
		},
		Selected: 0,
		Filter:   "be",
	}
	_, _, label := lv.RowCount()
	assert.Contains(t, label, "f:be")
}

func TestListViewModel_Render(t *testing.T) {
	schema := &DisplaySchema{
		List: &ListDisplayConfig{TitleField: "name"},
	}
	data := []interface{}{
		map[string]interface{}{"name": "alpha"},
		map[string]interface{}{"name": "beta"},
	}
	lv := buildListViewModel(data, schema, 80, 24)
	require.NotNil(t, lv)

	content := lv.Render(80, 20, true)
	assert.Contains(t, content, "alpha")
	assert.Contains(t, content, "beta")
}

func TestListViewModel_Update(t *testing.T) {
	lv := &ListViewModel{}
	result, cmd := lv.Update(nil)
	assert.Equal(t, lv, result)
	assert.Nil(t, cmd)
}

// ---------------------------------------------------------------------------
// DetailViewModel — CustomView methods
// ---------------------------------------------------------------------------

func TestDetailViewModel_Title(t *testing.T) {
	dv := &DetailViewModel{TitleText: "my-resource"}
	assert.Equal(t, "my-resource", dv.Title())
}

func TestDetailViewModel_Title_Nil(t *testing.T) {
	var dv *DetailViewModel
	assert.Equal(t, "", dv.Title())
}

func TestDetailViewModel_Title_Empty(t *testing.T) {
	dv := &DetailViewModel{}
	assert.Equal(t, "", dv.Title())
}

func TestDetailViewModel_FooterBar(t *testing.T) {
	dv := &DetailViewModel{}
	assert.Equal(t, "", dv.FooterBar())
}

func TestDetailViewModel_HandlesSearch(t *testing.T) {
	dv := &DetailViewModel{}
	assert.False(t, dv.HandlesSearch())
}

func TestDetailViewModel_Init(t *testing.T) {
	dv := &DetailViewModel{}
	assert.Nil(t, dv.Init())
}

func TestDetailViewModel_SearchTitle(t *testing.T) {
	dv := &DetailViewModel{}
	assert.Equal(t, "", dv.SearchTitle())
}

func TestDetailViewModel_FlashMessage(t *testing.T) {
	dv := &DetailViewModel{}
	msg, isErr := dv.FlashMessage()
	assert.Equal(t, "", msg)
	assert.False(t, isErr)
}

func TestDetailViewModel_RowCount(t *testing.T) {
	dv := &DetailViewModel{}
	count, sel, label := dv.RowCount()
	assert.Equal(t, 1, count)
	assert.Equal(t, 1, sel)
	assert.Equal(t, "detail", label)
}

func TestDetailViewModel_Update(t *testing.T) {
	dv := &DetailViewModel{}
	result, cmd := dv.Update(nil)
	assert.Equal(t, dv, result)
	assert.Nil(t, cmd)
}

func TestDetailViewModel_Render(t *testing.T) {
	schema := &DisplaySchema{
		Detail: &DetailDisplayConfig{
			TitleField: "name",
			Sections: []DetailSection{
				{Title: "Info", Fields: []string{"name", "region"}},
			},
		},
	}
	obj := map[string]interface{}{
		"name":   "my-vm",
		"region": "us-east-1",
	}
	dv := buildDetailViewModel(obj, schema, 80, 24)
	require.NotNil(t, dv)

	content := dv.Render(80, 20, true)
	assert.Contains(t, content, "us-east-1")
	// Title field ("name") is promoted to the border, not repeated in content.
	assert.Equal(t, "my-vm", dv.Title())
}

// ---------------------------------------------------------------------------
// StatusViewModel — CustomView methods
// ---------------------------------------------------------------------------

func TestStatusViewModel_CustomView_Title(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	assert.Equal(t, "Sign in to Entra", sv.Title())
}

func TestStatusViewModel_CustomView_Render(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	content := sv.Render(80, 20, true)
	assert.Contains(t, content, "Already authenticated")
}

func TestStatusViewModel_CustomView_RowCount(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	count, sel, label := sv.RowCount()
	assert.Equal(t, 1, count)
	assert.Equal(t, 1, sel)
	assert.Equal(t, "status", label)
}

func TestStatusViewModel_CustomView_FooterBar(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	bar := sv.FooterBar()
	assert.Contains(t, bar, "quit")
}

func TestStatusViewModel_CustomView_FlashMessage_None(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	msg, isErr := sv.FlashMessage()
	assert.Equal(t, "", msg)
	assert.False(t, isErr)
}

func TestStatusViewModel_CustomView_FlashMessage_Success(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	sv.FlashMsg = "✓ Copied to clipboard"
	msg, isErr := sv.FlashMessage()
	assert.Equal(t, "✓ Copied to clipboard", msg)
	assert.False(t, isErr, "success flash should NOT be an error")
}

func TestStatusViewModel_CustomView_FlashMessage_Error(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	sv.FlashMsg = "⚠ Copy failed: no clipboard"
	msg, isErr := sv.FlashMessage()
	assert.Equal(t, "⚠ Copy failed: no clipboard", msg)
	assert.True(t, isErr, "warning flash should be an error")
}

func TestStatusViewModel_CustomView_SearchTitle(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	assert.Equal(t, "", sv.SearchTitle())
}

func TestStatusViewModel_CustomView_HandlesSearch(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)

	assert.False(t, sv.HandlesSearch())
}

// ---------------------------------------------------------------------------
// customSearchTitle
// ---------------------------------------------------------------------------

func TestCustomSearchTitle_NoCustomView(t *testing.T) {
	m := &Model{ViewMode: ""}
	assert.Equal(t, "", m.customSearchTitle())
}

func TestCustomSearchTitle_ListDefault(t *testing.T) {
	m := &Model{
		ViewMode:      "list",
		ListViewState: &ListViewModel{},
		ListPanelMode: ListPanelModeSearch,
	}
	// Default search mode → empty (keep "Search" label)
	assert.Equal(t, "", m.customSearchTitle())
}

func TestCustomSearchTitle_ListFilter(t *testing.T) {
	m := &Model{
		ViewMode:      "list",
		ListViewState: &ListViewModel{},
		ListPanelMode: ListPanelModeFilter,
	}
	assert.Equal(t, "Filter", m.customSearchTitle())
}

func TestCustomSearchTitle_DetailView(t *testing.T) {
	m := &Model{
		ViewMode:        "detail",
		DetailViewState: &DetailViewModel{},
	}
	// Detail doesn't provide a search title.
	assert.Equal(t, "", m.customSearchTitle())
}

func TestCustomSearchTitle_StatusView(t *testing.T) {
	data := testStatusData()
	schema := testStatusSchema()
	sv := buildStatusViewModel(data, schema, KeyModeVim, true, nil, 80, 24)
	m := &Model{
		ViewMode:        "status",
		StatusViewState: sv,
	}
	assert.Equal(t, "", m.customSearchTitle())
}
