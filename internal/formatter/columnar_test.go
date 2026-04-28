package formatter

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderColumnarTable(t *testing.T) {
	t.Run("basic render", func(t *testing.T) {
		columns := []string{"name", "age"}
		rows := [][]string{
			{"Alice", "30"},
			{"Bob", "25"},
		}

		result := RenderColumnarTable(columns, rows, ColumnarOptions{
			NoColor:        true,
			RowNumberStyle: "numbered",
		})

		require.NotEmpty(t, result)
		lines := strings.Split(strings.TrimSpace(result), "\n")
		require.GreaterOrEqual(t, len(lines), 4) // header + separator + 2 rows

		// Check header contains column names
		assert.Contains(t, lines[0], "#")
		assert.Contains(t, lines[0], "name")
		assert.Contains(t, lines[0], "age")

		// Check data rows contain numbered indices
		assert.Contains(t, lines[2], "1")
		assert.Contains(t, lines[2], "Alice")
		assert.Contains(t, lines[2], "30")
		assert.Contains(t, lines[3], "2")
		assert.Contains(t, lines[3], "Bob")
	})

	t.Run("index style", func(t *testing.T) {
		columns := []string{"value"}
		rows := [][]string{{"a"}, {"b"}}

		result := RenderColumnarTable(columns, rows, ColumnarOptions{
			NoColor:        true,
			RowNumberStyle: "index",
		})

		assert.Contains(t, result, "[0]")
		assert.Contains(t, result, "[1]")
	})

	t.Run("bullet style", func(t *testing.T) {
		columns := []string{"item"}
		rows := [][]string{{"first"}, {"second"}}

		result := RenderColumnarTable(columns, rows, ColumnarOptions{
			NoColor:        true,
			RowNumberStyle: "bullet",
		})

		assert.Contains(t, result, "•")
	})

	t.Run("no row numbers", func(t *testing.T) {
		columns := []string{"name"}
		rows := [][]string{{"Alice"}}

		result := RenderColumnarTable(columns, rows, ColumnarOptions{
			NoColor:        true,
			RowNumberStyle: "none",
		})

		// Should not contain row number indicators
		assert.NotContains(t, result, "#")
		assert.Contains(t, result, "name")
		assert.Contains(t, result, "Alice")
	})

	t.Run("hidden columns", func(t *testing.T) {
		columns := []string{"id", "name", "secret"}
		rows := [][]string{
			{"1", "Alice", "pass123"},
		}

		result := RenderColumnarTable(columns, rows, ColumnarOptions{
			NoColor:        true,
			RowNumberStyle: "numbered",
			HiddenColumns:  []string{"secret"},
		})

		assert.Contains(t, result, "name")
		assert.NotContains(t, result, "secret")
		assert.NotContains(t, result, "pass123")
	})

	t.Run("empty columns returns empty", func(t *testing.T) {
		result := RenderColumnarTable([]string{}, [][]string{}, ColumnarOptions{})
		assert.Empty(t, result)
	})

	t.Run("empty rows returns empty", func(t *testing.T) {
		result := RenderColumnarTable([]string{"a"}, [][]string{}, ColumnarOptions{})
		assert.Empty(t, result)
	})
}

func TestFilterColumns(t *testing.T) {
	columns := []string{"a", "b", "c"}
	rows := [][]string{
		{"1", "2", "3"},
		{"4", "5", "6"},
	}

	t.Run("no hidden", func(t *testing.T) {
		cols, data := filterColumns(columns, rows, nil)
		assert.Equal(t, columns, cols)
		assert.Equal(t, rows, data)
	})

	t.Run("hide middle column", func(t *testing.T) {
		cols, data := filterColumns(columns, rows, []string{"b"})
		assert.Equal(t, []string{"a", "c"}, cols)
		assert.Equal(t, [][]string{{"1", "3"}, {"4", "6"}}, data)
	})

	t.Run("hide all columns", func(t *testing.T) {
		cols, data := filterColumns(columns, rows, []string{"a", "b", "c"})
		assert.Empty(t, cols)
		assert.Len(t, data, 2) // rows still present but empty
	})
}

func TestCalculateColumnWidths(t *testing.T) {
	t.Run("fits within width", func(t *testing.T) {
		columns := []string{"name", "id"}
		rows := [][]string{{"Alice", "123"}}

		widths := calculateColumnWidths(columns, rows, 100, nil)
		require.Len(t, widths, 2)
		// Widths should be at least header width
		assert.GreaterOrEqual(t, widths[0], 4) // "name"
		assert.GreaterOrEqual(t, widths[1], 2) // "id"
	})

	t.Run("expands for data", func(t *testing.T) {
		columns := []string{"x"}
		rows := [][]string{{"very long value here"}}

		widths := calculateColumnWidths(columns, rows, 100, nil)
		require.Len(t, widths, 1)
		// Should expand to fit data (or hit cap)
		assert.GreaterOrEqual(t, widths[0], 1)
	})
}

func TestCalculateColumnWidths_WithHints(t *testing.T) {
	t.Run("MaxWidth cap applied", func(t *testing.T) {
		columns := []string{"short", "long_col"}
		rows := [][]string{{"hi", "a very long value that should be capped"}}
		hints := []ColumnHint{
			{}, // short - no hint
			{MaxWidth: 10},
		}

		widths := calculateColumnWidths(columns, rows, 100, hints)
		require.Len(t, widths, 2)
		assert.LessOrEqual(t, widths[1], 10, "long_col should be capped at MaxWidth 10")
	})

	t.Run("priority-based shrinking", func(t *testing.T) {
		columns := []string{"important", "unimportant"}
		// Both columns have long data (30 chars each)
		rows := [][]string{{
			"012345678901234567890123456789",
			"012345678901234567890123456789",
		}}
		hints := []ColumnHint{
			{Priority: 10}, // important
			{Priority: 0},  // unimportant
		}

		// Available width is much less than needed (30+30+2sep = 62, give only 40)
		widths := calculateColumnWidths(columns, rows, 40, hints)
		require.Len(t, widths, 2)
		// Important column should be wider than unimportant
		assert.Greater(t, widths[0], widths[1],
			"higher-priority column should retain more width")
	})

	t.Run("flex column absorbs surplus", func(t *testing.T) {
		columns := []string{"status", "message"}
		rows := [][]string{
			{"ok", "short msg"},
		}
		hints := []ColumnHint{
			{},           // status - fixed
			{Flex: true}, // message - flex
		}

		// Natural widths: status=6("status"), message=9("short msg")
		// Total natural = 6+9 = 15, seps = 2, so 17 needed.
		// Give 60 chars available → 43 surplus should go to message.
		widths := calculateColumnWidths(columns, rows, 60, hints)
		require.Len(t, widths, 2)
		assert.Equal(t, 6, widths[0], "fixed column stays at natural width")
		assert.Equal(t, 52, widths[1], "flex column absorbs remaining space (60-2sep-6)")
	})

	t.Run("flex column expands beyond MaxWidth", func(t *testing.T) {
		columns := []string{"id", "description"}
		rows := [][]string{
			{"1", "short"},
		}
		hints := []ColumnHint{
			{},                         // id - fixed
			{Flex: true, MaxWidth: 30}, // description - flex
		}

		// Natural: id=2, description=11("description" header)
		// MaxWidth caps initial size to 30, but flex expands beyond that.
		// Give 100 chars → flex gets 100 - 2sep - 2 = 96.
		widths := calculateColumnWidths(columns, rows, 100, hints)
		require.Len(t, widths, 2)
		assert.Equal(t, 2, widths[0], "fixed column stays at natural width")
		assert.Equal(t, 96, widths[1], "flex column expands beyond MaxWidth to fill space")
	})

	t.Run("multiple flex columns split surplus", func(t *testing.T) {
		columns := []string{"a", "b", "c"}
		rows := [][]string{{"x", "y", "z"}}
		hints := []ColumnHint{
			{},           // a - fixed
			{Flex: true}, // b - flex
			{Flex: true}, // c - flex
		}

		// Natural: a=1, b=1, c=1, seps=4 → 7 total. Give 27 → 20 surplus
		// split between b and c.
		widths := calculateColumnWidths(columns, rows, 27, hints)
		require.Len(t, widths, 3)
		assert.Equal(t, 1, widths[0], "fixed column unchanged")
		total := widths[1] + widths[2]
		assert.Equal(t, 22, total, "flex columns absorb all surplus")
	})

	t.Run("no flex no expansion", func(t *testing.T) {
		columns := []string{"name", "age"}
		rows := [][]string{{"Alice", "30"}}
		hints := []ColumnHint{
			{Priority: 5},
			{Priority: 5},
		}

		// Natural: name=5("Alice"), age=3("age"), seps=2 → 10 total.
		// Give 100 chars → no expansion since no flex columns.
		widths := calculateColumnWidths(columns, rows, 100, hints)
		require.Len(t, widths, 2)
		assert.Equal(t, 5, widths[0], "non-flex stays at natural width")
		assert.Equal(t, 3, widths[1], "non-flex stays at natural width")
	})
}

func TestHasFlexColumn(t *testing.T) {
	t.Run("no hints", func(t *testing.T) {
		assert.False(t, HasFlexColumn(nil))
	})

	t.Run("no flex", func(t *testing.T) {
		hints := map[string]ColumnHint{
			"a": {MaxWidth: 10},
			"b": {Priority: 5},
		}
		assert.False(t, HasFlexColumn(hints))
	})

	t.Run("has flex", func(t *testing.T) {
		hints := map[string]ColumnHint{
			"a": {MaxWidth: 10},
			"b": {Flex: true},
		}
		assert.True(t, HasFlexColumn(hints))
	})

	t.Run("hidden flex column ignored", func(t *testing.T) {
		hints := map[string]ColumnHint{
			"a": {MaxWidth: 10},
			"b": {Flex: true, Hidden: true},
		}
		assert.False(t, HasFlexColumn(hints), "hidden flex column should be ignored")
	})
}

func TestColumnarOverhead(t *testing.T) {
	t.Run("single column no row numbers", func(t *testing.T) {
		// Just borders: 2
		assert.Equal(t, 2, ColumnarOverhead(1, false, 0))
	})

	t.Run("three columns no row numbers", func(t *testing.T) {
		// 2 borders + 2 separators * 2 = 6
		assert.Equal(t, 6, ColumnarOverhead(3, false, 0))
	})

	t.Run("two columns with row numbers", func(t *testing.T) {
		// 2 borders + 1 sep between cols + rowNumWidth(3 for "10"+2pad=4) + 1 sep for rowNum
		// = 2 + 2 + 4 + 2 = 10
		assert.Equal(t, 10, ColumnarOverhead(2, true, 10))
	})
}

func TestShrinkProportional(t *testing.T) {
	t.Run("narrow columns preserved while wide columns shrink", func(t *testing.T) {
		// 3 narrow (7 each) + 1 wide (40). Total = 61. usableWidth = 40.
		// Narrow columns should stay near natural size; wide column absorbs shrink.
		widths := shrinkProportional([]int{7, 7, 7, 40}, 40)
		total := totalUsed(widths)
		assert.Equal(t, 40, total, "total must equal usableWidth")

		// Narrow columns should keep their natural width (7) since they fit
		// within a fair share of 10.
		for i := 0; i < 3; i++ {
			assert.Equal(t, 7, widths[i], "narrow column %d should be preserved", i)
		}
		// Wide column gets the remainder
		assert.Equal(t, 19, widths[3], "wide column absorbs the shrink")
	})

	t.Run("lock guard prevents starvation of wide column", func(t *testing.T) {
		// Regression test: [7,7,7,40] at usableWidth=28.
		// Without the guard, locking [7,7,7] leaves only 7 for the wide column
		// (below minReadableWidth=8). The guard should detect this and fall back
		// to proportional shrink across all columns.
		widths := shrinkProportional([]int{7, 7, 7, 40}, 28)
		total := totalUsed(widths)
		assert.Equal(t, 28, total, "total must equal usableWidth")
		assert.GreaterOrEqual(t, widths[3], minReadableWidth,
			"wide column must stay above minReadableWidth after lock guard")
	})

	t.Run("all columns fit within usableWidth", func(t *testing.T) {
		widths := shrinkProportional([]int{5, 5, 5}, 30)
		assert.Equal(t, []int{5, 5, 5}, widths, "no shrinking needed")
	})

	t.Run("defaultMaxColWidth caps excessive widths", func(t *testing.T) {
		widths := shrinkProportional([]int{100, 10}, 60)
		total := totalUsed(widths)
		assert.Equal(t, 50, total, "total should equal usableWidth after cap+shrink")
		assert.LessOrEqual(t, widths[0], defaultMaxColWidth,
			"wide column should be capped at defaultMaxColWidth initially")
	})
}

func TestShrinkByPriority(t *testing.T) {
	t.Run("shrinks lowest priority first", func(t *testing.T) {
		widths := []int{20, 20, 20} // total=60
		hints := []ColumnHint{
			{Priority: 10}, // a
			{Priority: 5},  // b
			{Priority: 0},  // c
		}

		result := shrinkByPriority(widths, 45, hints)
		// Need to shed 15 from total 60 to reach 45
		// c (priority 0) should shrink first (20→5, can shed 17)
		assert.Equal(t, 20, result[0], "highest priority should keep its width")
		assert.Equal(t, 20, result[1], "mid priority should keep width when low can absorb")
		assert.Equal(t, 5, result[2], "lowest priority should shrink most")
	})

	t.Run("spreads across columns when lowest cant absorb all", func(t *testing.T) {
		widths := []int{20, 10, 10} // total=40
		hints := []ColumnHint{
			{Priority: 10}, // high
			{Priority: 0},  // low1
			{Priority: 0},  // low2
		}

		result := shrinkByPriority(widths, 20, hints)
		// Need to shed 20. low1 can shed 7 (10→3), low2 can shed 7 (10→3), high sheds 6 (20→14)
		assert.Equal(t, 3, result[1], "low1 should shrink to min")
		assert.Equal(t, 3, result[2], "low2 should shrink to min")
		assert.Equal(t, 14, result[0], "high should absorb remaining excess")
	})

	t.Run("no shrink needed", func(t *testing.T) {
		widths := []int{10, 10}
		result := shrinkByPriority(widths, 30, nil)
		assert.Equal(t, 10, result[0])
		assert.Equal(t, 10, result[1])
	})
}

func TestIsColumnarReadable(t *testing.T) {
	t.Run("readable at wide width", func(t *testing.T) {
		columns := []string{"name", "age", "city"}
		rows := [][]string{
			{"Alice", "30", "New York"},
			{"Bob", "25", "London"},
		}
		assert.True(t, IsColumnarReadable(columns, rows, 120, nil, IsColumnarReadableOpts{}))
	})

	t.Run("unreadable when many columns squeezed", func(t *testing.T) {
		columns := []string{"name", "email", "address", "phone", "company", "department", "title", "country"}
		rows := [][]string{
			{"Alice Johnson", "alice@example.com", "123 Main St", "555-1234", "Acme Corp", "Engineering", "Senior Dev", "United States"},
		}
		// At 40 chars wide, 8 substantial columns cannot possibly fit readably
		assert.False(t, IsColumnarReadable(columns, rows, 40, nil, IsColumnarReadableOpts{}))
	})

	t.Run("naturally narrow columns are fine", func(t *testing.T) {
		// Columns whose content is shorter than minReadableWidth should not trigger unreadable
		columns := []string{"ok", "yn"}
		rows := [][]string{
			{"yes", "no"},
			{"yes", "yes"},
		}
		assert.True(t, IsColumnarReadable(columns, rows, 20, nil, IsColumnarReadableOpts{}))
	})

	t.Run("empty columns readable", func(t *testing.T) {
		assert.True(t, IsColumnarReadable(nil, nil, 80, nil, IsColumnarReadableOpts{}))
	})

	t.Run("hidden columns make table fit", func(t *testing.T) {
		columns := []string{"name", "email", "address", "phone", "company", "department"}
		rows := [][]string{
			{"Alice Johnson", "alice@example.com", "123 Main St", "555-1234", "Acme Corp", "Engineering"},
		}
		// Unreadable at 50 with all columns
		assert.False(t, IsColumnarReadable(columns, rows, 50, nil, IsColumnarReadableOpts{}))
		// Readable when most columns hidden
		assert.True(t, IsColumnarReadable(columns, rows, 50, nil, IsColumnarReadableOpts{
			HiddenColumns: []string{"address", "phone", "company", "department"},
		}))
	})

	t.Run("row numbers reduce available width", func(t *testing.T) {
		// Use 5 columns with 10-char data. Natural width = 10*5 = 50, plus 4 seps = 58 min.
		// The key is to find a width where it's readable without row nums but not with.
		columns := []string{"column_aaa", "column_bbb", "column_ccc", "column_ddd", "column_eee"}
		rows := [][]string{
			{"value_1234", "value_5678", "value_9abc", "value_defg", "value_hijk"},
		}
		// At 50 chars: without row nums, columns get shrunk but stay above 8
		// With row nums (~5 chars including sep), available space shrinks further
		assert.True(t, IsColumnarReadable(columns, rows, 50, nil, IsColumnarReadableOpts{RowNumberStyle: "none"}))
		// At tight 45 chars with row numbers, columns must shrink below 8
		assert.False(t, IsColumnarReadable(columns, rows, 45, nil, IsColumnarReadableOpts{RowNumberStyle: "numbered"}))
		// With ample width, always readable
		assert.True(t, IsColumnarReadable(columns, rows, 70, nil, IsColumnarReadableOpts{RowNumberStyle: "numbered"}))
	})

	t.Run("MaxWidth hint below minReadable does not flag column as unreadable", func(t *testing.T) {
		// A column with a long header name ("severity"=8) but MaxWidth=5 (from
		// an enum like ["error","warn","info"]) should not be treated as unreadable.
		// The MaxWidth cap means the column is designed to be narrow.
		columns := []string{"severity", "message"}
		rows := [][]string{
			{"error", "something went wrong"},
			{"warn", "check this out"},
		}
		hints := map[string]ColumnHint{
			"severity": {MaxWidth: 5, Priority: 10},
			"message":  {Priority: 15},
		}
		// At 60 chars both columns fit; severity gets capped to 5 which is fine
		assert.True(t, IsColumnarReadable(columns, rows, 60, hints, IsColumnarReadableOpts{}))
	})

	t.Run("flex column never triggers unreadable", func(t *testing.T) {
		// A flex column with very wide data should not cause list fallback.
		// Flex columns accept whatever width they get by design.
		columns := []string{"severity", "message"}
		rows := [][]string{
			{"error", "validating https://scafctl.dev/api/v1/clusters/prod-east-1/nodes/worker-07 against schema version 2.4.1"},
			{"warn", "certificate expiry approaching for endpoint https://scafctl.dev/api/v1/auth/tokens/refresh"},
		}
		hints := map[string]ColumnHint{
			"severity": {MaxWidth: 8, Priority: 10},
			"message":  {Flex: true, Priority: 5},
		}
		// At 80 chars, the 128-char message would normally be unreadable,
		// but since it's Flex, the table should still be considered readable.
		assert.True(t, IsColumnarReadable(columns, rows, 80, hints, IsColumnarReadableOpts{}))
		// Also works at wider widths
		assert.True(t, IsColumnarReadable(columns, rows, 120, hints, IsColumnarReadableOpts{}))
		assert.True(t, IsColumnarReadable(columns, rows, 160, hints, IsColumnarReadableOpts{}))
	})

	t.Run("flex column with MaxWidth never triggers unreadable", func(t *testing.T) {
		columns := []string{"severity", "message"}
		rows := [][]string{
			{"error", "validating https://scafctl.dev/api/v1/clusters/prod-east-1"},
		}
		hints := map[string]ColumnHint{
			"severity": {MaxWidth: 8, Priority: 10},
			"message":  {Flex: true, MaxWidth: 60, Priority: 5},
		}
		assert.True(t, IsColumnarReadable(columns, rows, 80, hints, IsColumnarReadableOpts{}))
	})

	t.Run("flex-only no MaxWidth does not squeeze fixed columns", func(t *testing.T) {
		// Simulates schema-parsed columns where unbounded strings get
		// Flex: true with no MaxWidth. The flex column's wide content
		// must not inflate totalNeeded and squeeze fixed columns below
		// minReadableWidth.
		columns := []string{"name", "status", "description"}
		rows := [][]string{
			{"prod-east-1", "healthy", "primary production cluster serving east-coast traffic with 47 worker nodes across 3 availability zones"},
			{"staging-west", "degraded", "staging environment for west-coast deployment pipeline validation and integration testing"},
		}
		hints := map[string]ColumnHint{
			"name":        {Priority: 20},
			"status":      {Priority: 15, MaxWidth: 10},
			"description": {Flex: true, Priority: 5},
		}
		// At 80 chars, fixed columns (name=12, status=10) fit easily.
		// The 115-char description is flex and should not cause fallback.
		assert.True(t, IsColumnarReadable(columns, rows, 80, hints, IsColumnarReadableOpts{}),
			"flex-only column with wide content should not trigger list fallback")
		// Even at narrow widths, flex absorbs the squeeze
		assert.True(t, IsColumnarReadable(columns, rows, 50, hints, IsColumnarReadableOpts{}),
			"flex-only column should keep table readable at narrow width")
	})
}

func TestColumnsToDropForReadability(t *testing.T) {
	t.Run("already readable returns nil", func(t *testing.T) {
		columns := []string{"name", "age"}
		rows := [][]string{{"Alice", "30"}}
		result := ColumnsToDropForReadability(columns, rows, 120, nil, IsColumnarReadableOpts{})
		assert.Nil(t, result)
	})

	t.Run("drops lowest priority column first", func(t *testing.T) {
		columns := []string{"name", "email", "address", "phone"}
		rows := [][]string{
			{"Alice Johnson", "alice@example.com", "123 Main St, Apt 4", "555-123-4567"},
		}
		hints := map[string]ColumnHint{
			"name":    {Priority: 20},
			"email":   {Priority: 10},
			"address": {Priority: 2},
			"phone":   {Priority: 1},
		}
		result := ColumnsToDropForReadability(columns, rows, 40, hints, IsColumnarReadableOpts{})
		assert.NotNil(t, result)
		// Lowest priority columns should be dropped first
		assert.Contains(t, result, "phone")
	})

	t.Run("drops multiple columns if needed", func(t *testing.T) {
		columns := []string{"name", "email", "address", "phone", "company", "department"}
		rows := [][]string{
			{"Alice Johnson", "alice@example.com", "123 Main St", "555-1234", "Acme Corp", "Engineering"},
		}
		hints := map[string]ColumnHint{
			"name":       {Priority: 30},
			"email":      {Priority: 20},
			"address":    {Priority: 5},
			"phone":      {Priority: 4},
			"company":    {Priority: 3},
			"department": {Priority: 2},
		}
		result := ColumnsToDropForReadability(columns, rows, 45, hints, IsColumnarReadableOpts{})
		assert.NotNil(t, result)
		assert.Greater(t, len(result), 1, "should drop multiple low-priority columns")
		// High priority columns should not be dropped
		assert.NotContains(t, result, "name")
		assert.NotContains(t, result, "email")
	})

	t.Run("returns nil when even dropping all leaves unreadable", func(t *testing.T) {
		columns := []string{"very_long_column_name_a", "very_long_column_name_b"}
		rows := [][]string{
			{"some long value here", "another long value here"},
		}
		// Width so narrow that even 1 column can't fit readably
		result := ColumnsToDropForReadability(columns, rows, 5, nil, IsColumnarReadableOpts{})
		assert.Nil(t, result)
	})

	t.Run("respects already hidden columns", func(t *testing.T) {
		columns := []string{"name", "email", "address", "phone"}
		rows := [][]string{
			{"Alice Johnson", "alice@example.com", "123 Main St", "555-1234"},
		}
		hints := map[string]ColumnHint{
			"name":    {Priority: 20},
			"email":   {Priority: 10},
			"address": {Priority: 2},
			"phone":   {Priority: 1},
		}
		opts := IsColumnarReadableOpts{HiddenColumns: []string{"phone"}}
		result := ColumnsToDropForReadability(columns, rows, 40, hints, opts)
		// phone is already hidden, so it should not appear in result
		if result != nil {
			assert.NotContains(t, result, "phone")
		}
	})

	t.Run("never drops below one visible column", func(t *testing.T) {
		columns := []string{"name", "email"}
		rows := [][]string{
			{"Alice Johnson", "alice@example.com"},
		}
		hints := map[string]ColumnHint{
			"name":  {Priority: 10},
			"email": {Priority: 1},
		}
		// Very narrow: even after dropping email, name alone may not be readable
		result := ColumnsToDropForReadability(columns, rows, 3, hints, IsColumnarReadableOpts{})
		// Should not drop both columns
		if result != nil {
			assert.Less(t, len(result), len(columns), "should keep at least one column")
		}
	})
}

func TestRenderColumnarTable_WithHints(t *testing.T) {
	columns := []string{"name", "value"}
	rows := [][]string{
		{"Alice", "123"},
		{"Bob", "45678"},
	}

	t.Run("right-align numeric column", func(t *testing.T) {
		result := RenderColumnarTable(columns, rows, ColumnarOptions{
			NoColor:        true,
			TotalWidth:     40,
			RowNumberStyle: "none",
			ColumnHints: map[string]ColumnHint{
				"value": {Align: "right"},
			},
		})
		assert.Contains(t, result, "name")
		assert.Contains(t, result, "value")
		// Right-aligned "123" in a 5-wide column should have leading spaces
		lines := strings.Split(result, "\n")
		found := false
		for _, line := range lines {
			if strings.Contains(line, "Alice") && strings.Contains(line, "123") {
				idx := strings.Index(line, "123")
				if idx > 0 && line[idx-1] == ' ' {
					found = true
				}
			}
		}
		assert.True(t, found, "value column should be right-aligned with leading space for '123'")
	})

	t.Run("MaxWidth caps column", func(t *testing.T) {
		longRows := [][]string{
			{"A very long name that exceeds the cap", "val"},
		}
		result := RenderColumnarTable(columns, longRows, ColumnarOptions{
			NoColor:        true,
			TotalWidth:     100,
			RowNumberStyle: "none",
			ColumnHints: map[string]ColumnHint{
				"name": {MaxWidth: 10},
			},
		})
		lines := strings.Split(result, "\n")
		for _, line := range lines {
			if strings.Contains(line, "val") {
				assert.NotContains(t, line, "A very long name that exceeds the cap")
			}
		}
	})

	t.Run("DisplayName renames header", func(t *testing.T) {
		result := RenderColumnarTable(columns, rows, ColumnarOptions{
			NoColor:        true,
			TotalWidth:     40,
			RowNumberStyle: "none",
			ColumnHints: map[string]ColumnHint{
				"name": {DisplayName: "Full Name"},
			},
		})
		assert.Contains(t, result, "Full Name")
		assert.NotContains(t, result, "name  ")
	})

	t.Run("DisplayName with MaxWidth applies cap to renamed column", func(t *testing.T) {
		longRows := [][]string{
			{"A really long name here", "val"},
		}
		result := RenderColumnarTable(columns, longRows, ColumnarOptions{
			NoColor:        true,
			TotalWidth:     100,
			RowNumberStyle: "none",
			ColumnHints: map[string]ColumnHint{
				"name": {DisplayName: "Full Name", MaxWidth: 10},
			},
		})
		assert.Contains(t, result, "Full Name")
		lines := strings.Split(result, "\n")
		for _, line := range lines {
			if strings.Contains(line, "val") {
				// The full long value should be truncated
				assert.NotContains(t, line, "A really long name here",
					"MaxWidth should cap the renamed column")
			}
		}
	})

	t.Run("DisplayName with Align applies alignment to renamed column", func(t *testing.T) {
		result := RenderColumnarTable(columns, rows, ColumnarOptions{
			NoColor:        true,
			TotalWidth:     40,
			RowNumberStyle: "none",
			ColumnHints: map[string]ColumnHint{
				"value": {DisplayName: "Score", Align: "right"},
			},
		})
		assert.Contains(t, result, "Score")
		// Right-aligned "123" should have leading space
		lines := strings.Split(result, "\n")
		found := false
		for _, line := range lines {
			if strings.Contains(line, "Alice") && strings.Contains(line, "123") {
				idx := strings.Index(line, "123")
				if idx > 0 && line[idx-1] == ' ' {
					found = true
				}
			}
		}
		assert.True(t, found, "renamed column should still be right-aligned")
	})

	t.Run("hidden column with DisplayName is still hidden", func(t *testing.T) {
		result := RenderColumnarTable(columns, rows, ColumnarOptions{
			NoColor:        true,
			TotalWidth:     40,
			RowNumberStyle: "none",
			HiddenColumns:  []string{"name"},
			ColumnHints: map[string]ColumnHint{
				"name": {DisplayName: "Full Name"},
			},
		})
		assert.NotContains(t, result, "Full Name")
		assert.NotContains(t, result, "name")
		assert.Contains(t, result, "value")
	})
}

func TestDefaultTableFormatOptions(t *testing.T) {
	opts := DefaultTableFormatOptions()
	assert.Equal(t, "numbered", opts.ArrayStyle)
	assert.Equal(t, "auto", opts.ColumnarMode)
	assert.Empty(t, opts.ColumnOrder)
	assert.Empty(t, opts.HiddenColumns)
	assert.Empty(t, opts.SelectColumns)
}

func TestApplySelectColumns(t *testing.T) {
	t.Run("no select columns is no-op", func(t *testing.T) {
		opts := TableFormatOptions{}
		opts.ApplySelectColumns([]string{"a", "b", "c"})
		assert.Empty(t, opts.HiddenColumns)
		assert.Empty(t, opts.ColumnOrder)
	})

	t.Run("select hides non-selected", func(t *testing.T) {
		opts := TableFormatOptions{SelectColumns: []string{"a", "c"}}
		opts.ApplySelectColumns([]string{"a", "b", "c", "d"})
		assert.Contains(t, opts.HiddenColumns, "b")
		assert.Contains(t, opts.HiddenColumns, "d")
		assert.Equal(t, []string{"a", "c"}, opts.ColumnOrder)
	})

	t.Run("already hidden columns not duplicated", func(t *testing.T) {
		opts := TableFormatOptions{
			SelectColumns: []string{"a"},
			HiddenColumns: []string{"b"},
		}
		opts.ApplySelectColumns([]string{"a", "b", "c"})
		// b was already hidden, c is newly hidden
		count := 0
		for _, h := range opts.HiddenColumns {
			if h == "b" {
				count++
			}
		}
		assert.Equal(t, 1, count, "b should appear only once")
		assert.Contains(t, opts.HiddenColumns, "c")
	})
}

func TestEffectiveColumnOrder(t *testing.T) {
	t.Run("returns SelectColumns when set", func(t *testing.T) {
		opts := TableFormatOptions{
			ColumnOrder:   []string{"a", "b"},
			SelectColumns: []string{"x", "y"},
		}
		assert.Equal(t, []string{"x", "y"}, opts.EffectiveColumnOrder())
	})

	t.Run("returns ColumnOrder when no select", func(t *testing.T) {
		opts := TableFormatOptions{ColumnOrder: []string{"a", "b"}}
		assert.Equal(t, []string{"a", "b"}, opts.EffectiveColumnOrder())
	})

	t.Run("returns nil when both empty", func(t *testing.T) {
		opts := TableFormatOptions{}
		assert.Nil(t, opts.EffectiveColumnOrder())
	})
}

func TestCalculateNaturalColumnarWidth(t *testing.T) {
	columns := []string{"name", "age"}
	rows := [][]string{
		{"Alice", "30"},
		{"Bob", "25"},
	}

	w := CalculateNaturalColumnarWidth(columns, rows, false, 2)
	assert.Greater(t, w, 0)

	wWithRowNum := CalculateNaturalColumnarWidth(columns, rows, true, 2)
	assert.Greater(t, wWithRowNum, w, "row numbers should add width")
}

func TestCalculateNaturalColumnarWidthWithHints(t *testing.T) {
	columns := []string{"name", "description"}
	rows := [][]string{
		{"Alice", "A very long description that would normally be wide"},
	}

	wNoHints := CalculateNaturalColumnarWidthWithHints(columns, rows, false, 1, nil, nil)
	assert.Greater(t, wNoHints, 0)

	hints := map[string]ColumnHint{
		"description": {MaxWidth: 10},
	}
	wWithHints := CalculateNaturalColumnarWidthWithHints(columns, rows, false, 1, hints, nil)
	assert.Less(t, wWithHints, wNoHints, "MaxWidth hint should reduce natural width")

	wHidden := CalculateNaturalColumnarWidthWithHints(columns, rows, false, 1, nil, []string{"description"})
	assert.Less(t, wHidden, wNoHints, "hidden column should reduce width")
}

func TestCalculateNaturalColumnarWidth_Empty(t *testing.T) {
	assert.Equal(t, 0, CalculateNaturalColumnarWidth(nil, nil, false, 0))
	assert.Equal(t, 0, CalculateNaturalColumnarWidth([]string{}, nil, false, 0))
}

func TestColumnDropRendering(t *testing.T) {
	columns := []string{"user_id", "full_name", "score", "department"}
	rows := [][]string{
		{"u001", "Alice Smith", "95", "Engineering"},
		{"u002", "Bob Jones", "87", "Sales"},
		{"u003", "Carol White", "92", "Engineering"},
	}
	hints := map[string]ColumnHint{
		"user_id":    {Priority: 20, MaxWidth: 8, DisplayName: "ID"},
		"full_name":  {Priority: 18, DisplayName: "Name"},
		"score":      {Priority: 5, Align: "right", DisplayName: "Score"},
		"department": {Priority: 3, DisplayName: "Dept"},
	}

	t.Run("wide table shows all columns", func(t *testing.T) {
		opts := IsColumnarReadableOpts{RowNumberStyle: "none"}
		toDrop := ColumnsToDropForReadability(columns, rows, 80, hints, opts)
		assert.Nil(t, toDrop, "should not drop any columns at 80 width")

		result := RenderColumnarTable(columns, rows, ColumnarOptions{
			NoColor:        true,
			TotalWidth:     80,
			RowNumberStyle: "none",
			ColumnHints:    hints,
		})
		assert.Contains(t, result, "ID")
		assert.Contains(t, result, "Name")
		assert.Contains(t, result, "Score")
		assert.Contains(t, result, "Dept")
	})

	t.Run("narrow table drops low-priority columns", func(t *testing.T) {
		opts := IsColumnarReadableOpts{RowNumberStyle: "none"}
		toDrop := ColumnsToDropForReadability(columns, rows, 30, hints, opts)
		require.NotNil(t, toDrop, "should drop columns at 30 width")

		// Dropped columns should be lowest priority
		for _, col := range toDrop {
			assert.True(t, hints[col].Priority < 10,
				"dropped column %q (priority %d) should be low priority", col, hints[col].Priority)
		}

		// Render with dropped columns hidden
		result := RenderColumnarTable(columns, rows, ColumnarOptions{
			NoColor:        true,
			TotalWidth:     30,
			RowNumberStyle: "none",
			HiddenColumns:  toDrop,
			ColumnHints:    hints,
		})
		// High-priority columns survive
		assert.Contains(t, result, "Name")
		assert.Contains(t, result, "ID")
		// Low-priority columns are gone
		for _, col := range toDrop {
			assert.NotContains(t, result, hints[col].DisplayName,
				"dropped column %q should not appear in output", col)
		}
	})

	t.Run("explicit table override keeps all columns", func(t *testing.T) {
		// Simulates -o table: render all columns regardless of width
		result := RenderColumnarTable(columns, rows, ColumnarOptions{
			NoColor:        true,
			TotalWidth:     30,
			RowNumberStyle: "none",
			ColumnHints:    hints,
		})
		// All columns present even if truncated
		assert.Contains(t, result, "ID")
		assert.Contains(t, result, "Name")
	})
}

func TestColumnsToDropForReadability_EqualPriority(t *testing.T) {
	columns := []string{"alpha", "bravo", "charlie", "delta"}
	rows := [][]string{
		{"value_alpha", "value_bravo", "value_charlie", "value_delta"},
	}
	// No hints: all priorities are 0, tie-breaker should drop later columns first
	result := ColumnsToDropForReadability(columns, rows, 30, nil, IsColumnarReadableOpts{})
	require.NotNil(t, result)
	// delta (index 3) should be dropped before charlie (index 2), etc.
	assert.Equal(t, "delta", result[0], "highest index should be dropped first")
	if len(result) > 1 {
		assert.Equal(t, "charlie", result[1], "second highest index should be dropped second")
	}
}

func TestColumnsToDropForReadability_SingleColumn(t *testing.T) {
	// With only one visible column, there's nothing to drop
	columns := []string{"name"}
	rows := [][]string{
		{"Alice Johnson"},
	}
	result := ColumnsToDropForReadability(columns, rows, 3, nil, IsColumnarReadableOpts{})
	assert.Nil(t, result, "should not drop the only column")
}
