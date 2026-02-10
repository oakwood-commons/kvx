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
		hints := map[string]ColumnHint{
			"long_col": {MaxWidth: 10},
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
		hints := map[string]ColumnHint{
			"important":   {Priority: 10},
			"unimportant": {Priority: 0},
		}

		// Available width is much less than needed (30+30+2sep = 62, give only 40)
		widths := calculateColumnWidths(columns, rows, 40, hints)
		require.Len(t, widths, 2)
		// Important column should be wider than unimportant
		assert.Greater(t, widths[0], widths[1],
			"higher-priority column should retain more width")
	})
}

func TestShrinkByPriority(t *testing.T) {
	t.Run("shrinks lowest priority first", func(t *testing.T) {
		columns := []string{"a", "b", "c"}
		widths := []int{20, 20, 20} // total=60
		hints := map[string]ColumnHint{
			"a": {Priority: 10},
			"b": {Priority: 5},
			"c": {Priority: 0},
		}

		result := shrinkByPriority(columns, widths, 45, hints)
		// Need to shed 15 from total 60 to reach 45
		// c (priority 0) should shrink first (20→5, can shed 17)
		assert.Equal(t, 20, result[0], "highest priority should keep its width")
		assert.Equal(t, 20, result[1], "mid priority should keep width when low can absorb")
		assert.Equal(t, 5, result[2], "lowest priority should shrink most")
	})

	t.Run("spreads across columns when lowest cant absorb all", func(t *testing.T) {
		columns := []string{"high", "low1", "low2"}
		widths := []int{20, 10, 10} // total=40
		hints := map[string]ColumnHint{
			"high": {Priority: 10},
			"low1": {Priority: 0},
			"low2": {Priority: 0},
		}

		result := shrinkByPriority(columns, widths, 20, hints)
		// Need to shed 20. low1 can shed 7 (10→3), low2 can shed 7 (10→3), high sheds 6 (20→14)
		assert.Equal(t, 3, result[1], "low1 should shrink to min")
		assert.Equal(t, 3, result[2], "low2 should shrink to min")
		assert.Equal(t, 14, result[0], "high should absorb remaining excess")
	})

	t.Run("no shrink needed", func(t *testing.T) {
		columns := []string{"a", "b"}
		widths := []int{10, 10}
		result := shrinkByPriority(columns, widths, 30, nil)
		assert.Equal(t, 10, result[0])
		assert.Equal(t, 10, result[1])
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
}
