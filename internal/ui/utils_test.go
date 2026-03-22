package ui

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
)

func TestClampANSITextWidth_PlainText(t *testing.T) {
	assert.Equal(t, "hello", clampANSITextWidth("hello world", 5))
	assert.Equal(t, "hello world", clampANSITextWidth("hello world", 80))
}

func TestClampANSITextWidth_CSI(t *testing.T) {
	// CSI bold + text + reset: visible width is 5 ("hello")
	input := "\x1b[1mhello\x1b[0m world"
	result := clampANSITextWidth(input, 5)
	assert.Equal(t, "\x1b[1mhello\x1b[0m", result)
}

func TestClampANSITextWidth_OSCHyperlink(t *testing.T) {
	// OSC 8 hyperlink wrapping a URL — the escape sequences should be zero-width.
	url := "https://microsoft.entra.com/oauth/authorize?code=ABC123"
	hyperlink := "\x1b]8;;" + url + "\x1b\\" + url + "\x1b]8;;\x1b\\"

	// The visible width is just len(url) = 55.
	// With a max of 80, nothing should be clipped.
	result := clampANSITextWidth(hyperlink, 80)
	assert.Equal(t, hyperlink, result,
		"hyperlink should NOT be truncated when there is enough width")

	// With a max of 30, the visible URL should be truncated to 30 chars
	// but both OSC open and close sequences should be preserved.
	result30 := clampANSITextWidth(hyperlink, 30)
	assert.Contains(t, result30, "\x1b]8;;"+url+"\x1b\\",
		"opening hyperlink escape should be preserved")
	assert.Equal(t, 30, visibleWidth(result30),
		"visible width should be clamped to 30")
}

func TestClampANSITextWidth_OSCHyperlinkWithLabel(t *testing.T) {
	// "  URL: <hyperlink>" — label takes 7 visible chars.
	url := "https://microsoft.entra.com/oauth/authorize"
	line := "  URL: \x1b]8;;" + url + "\x1b\\" + url + "\x1b]8;;\x1b\\"

	// With width 60, the label (7) + URL (43) = 50, fits fine.
	result := clampANSITextWidth(line, 60)
	assert.Equal(t, line, result, "should not truncate when there is room")
}

func TestClampANSITextWidth_OSCWithBEL(t *testing.T) {
	// OSC terminated by BEL (0x07) instead of ST (ESC \).
	url := "https://example.com"
	hyperlink := "\x1b]8;;" + url + "\x07" + url + "\x1b]8;;\x07"

	result := clampANSITextWidth(hyperlink, 80)
	assert.Equal(t, hyperlink, result)
}

func TestClampANSITextWidth_Newline(t *testing.T) {
	result := clampANSITextWidth("abcde\nfghij", 3)
	assert.Equal(t, "abc\nfgh", result)
}

// visibleWidth returns the visible width of a string, ignoring ANSI escapes.
func visibleWidth(s string) int {
	return lipgloss.Width(s)
}

func TestRepeatToWidth(t *testing.T) {
	assert.Equal(t, "", repeatToWidth("x", 0))
	assert.Equal(t, "", repeatToWidth("x", -1))
	r := repeatToWidth("ab", 5)
	assert.Equal(t, 5, lipgloss.Width(r))
	// Whitespace-only fill defaults to space
	r2 := repeatToWidth(" ", 3)
	assert.Equal(t, "   ", r2)
}

func TestWrapPlainText(t *testing.T) {
	assert.Equal(t, "hello", wrapPlainText("hello", 0))
	assert.Equal(t, "hello world", wrapPlainText("hello world", 80))
	wrapped := wrapPlainText("hello world foo", 6)
	assert.Contains(t, wrapped, "hello")
	assert.Contains(t, wrapped, "world")
	// Blank lines preserved
	assert.Equal(t, "\n", wrapPlainText("\n", 80))
}

func TestIsCompositeNode(t *testing.T) {
	assert.True(t, isCompositeNode(map[string]interface{}{"a": 1}))
	assert.True(t, isCompositeNode([]interface{}{1, 2}))
	assert.False(t, isCompositeNode("string"))
	assert.False(t, isCompositeNode(42))
	assert.False(t, isCompositeNode(nil))
}

func TestPadANSIToWidth(t *testing.T) {
	assert.Equal(t, "hi   ", padANSIToWidth("hi", 5))
	assert.Equal(t, "hello", padANSIToWidth("hello", 3))
	assert.Equal(t, "hello", padANSIToWidth("hello", 5))
}

func TestAnsiVisibleWidth(t *testing.T) {
	assert.Equal(t, 5, ansiVisibleWidth("hello"))
	assert.Equal(t, 5, ansiVisibleWidth("\x1b[1mhello\x1b[0m"))
	assert.Equal(t, 0, ansiVisibleWidth(""))
}

func TestIntMax(t *testing.T) {
	assert.Equal(t, 5, intMax(3, 5))
	assert.Equal(t, 5, intMax(5, 3))
	assert.Equal(t, 5, intMax(5, 5))
}

func TestClampANSITextHeight(t *testing.T) {
	assert.Equal(t, "", clampANSITextHeight("hello\nworld", 0))
	assert.Equal(t, "hello", clampANSITextHeight("hello\nworld", 1))
	assert.Equal(t, "hello\nworld", clampANSITextHeight("hello\nworld", 5))
	assert.Equal(t, "", clampANSITextHeight("", 5))
	assert.Equal(t, "", clampANSITextHeight("\n\n", 5))
}

func TestIsSimpleScalarArray(t *testing.T) {
	assert.True(t, isSimpleScalarArray([]interface{}{"a", 1, true}))
	assert.True(t, isSimpleScalarArray([]interface{}{}))
	assert.False(t, isSimpleScalarArray([]interface{}{"a", map[string]interface{}{"b": 1}}))
	assert.False(t, isSimpleScalarArray([]interface{}{[]interface{}{1}}))
}

func TestLeftTruncate(t *testing.T) {
	assert.Equal(t, "", leftTruncate("hello", 0))
	assert.Equal(t, "hello", leftTruncate("hello", 10))
	assert.Equal(t, "llo", leftTruncate("hello", 3))
	assert.Equal(t, "o", leftTruncate("hello", 1))
}

func TestLeftTruncateANSI(t *testing.T) {
	assert.Equal(t, "", leftTruncateANSI("hello", 0))
	assert.Equal(t, "hello", leftTruncateANSI("hello", 10))
	assert.Equal(t, "llo", leftTruncateANSI("hello", 3))
	// With ANSI codes
	input := "\x1b[31mhello\x1b[0m"
	result := leftTruncateANSI(input, 3)
	assert.Equal(t, 3, ansiVisibleWidth(result))
}

func TestWindowTable(t *testing.T) {
	// Simple table with header, separator, and rows
	table := "HEADER\n------\nrow1\nrow2\nrow3\nrow4\nrow5\n"

	t.Run("fits", func(t *testing.T) {
		result, sel := windowTable(table, 0, 20)
		assert.Contains(t, result, "HEADER")
		assert.Equal(t, 0, sel)
	})

	t.Run("too tall", func(t *testing.T) {
		result, sel := windowTable(table, 0, 4)
		assert.Contains(t, result, "HEADER")
		_ = sel
	})

	t.Run("selected at end", func(t *testing.T) {
		result, _ := windowTable(table, 4, 4)
		assert.Contains(t, result, "HEADER")
	})

	t.Run("zero maxLines", func(t *testing.T) {
		result, _ := windowTable(table, 0, 0)
		assert.Equal(t, "", result)
	})

	t.Run("one maxLine", func(t *testing.T) {
		result, _ := windowTable(table, 0, 1)
		assert.Contains(t, result, "HEADER")
	})

	t.Run("empty", func(t *testing.T) {
		result, sel := windowTable("", 0, 5)
		assert.Equal(t, "", result)
		assert.Equal(t, 0, sel)
	})
}

func TestHighlightTableRow(t *testing.T) {
	table := "KEY    VALUE\n------\nname   test\nage    42\n"

	t.Run("noColor", func(t *testing.T) {
		result := highlightTableRow(table, 0, 40, true)
		assert.Contains(t, result, "name")
	})

	t.Run("withColor", func(t *testing.T) {
		result := highlightTableRow(table, 1, 40, false)
		assert.Contains(t, result, "age")
	})

	t.Run("too few lines", func(t *testing.T) {
		result := highlightTableRow("one\ntwo\n", 0, 40, true)
		assert.Equal(t, "one\ntwo\n", result)
	})

	t.Run("selected beyond rows", func(t *testing.T) {
		result := highlightTableRow(table, 100, 40, true)
		assert.Contains(t, result, "age")
	})
}

func TestRenderScalarBlock(t *testing.T) {
	result := renderScalarBlock("hello world", 40, true)
	assert.Contains(t, result, "VALUE")
	assert.Contains(t, result, "hello world")
}

func TestRenderScalarBlock_MinWidth(t *testing.T) {
	result := renderScalarBlock(42, 20, true)
	assert.Contains(t, result, "VALUE")
	assert.Contains(t, result, "42")
}

func TestSearchPrompt(t *testing.T) {
	result := searchPrompt(true)
	assert.NotEmpty(t, result)
}

func TestRenderSearchTable(t *testing.T) {
	hits := []searchHit{
		{FullPath: "a.b", Key: "name", Value: "test", Node: "test"},
		{FullPath: "a.c", Key: "age", Value: "42", Node: map[string]interface{}{"age": 42}},
	}
	result := renderSearchTable(hits, 10, 20, true)
	assert.Contains(t, result, "KEY")
	assert.Contains(t, result, "VALUE")
	assert.Contains(t, result, "test")
}

func TestRenderSearchTable_Defaults(t *testing.T) {
	hits := []searchHit{
		{FullPath: "x", Key: "", Value: "val", Node: "val"},
	}
	result := renderSearchTable(hits, 0, 0, true)
	assert.Contains(t, result, "KEY")
}

func TestContainsF1(t *testing.T) {
	assert.True(t, containsF1([]string{"<f1>"}))
	assert.True(t, containsF1([]string{"F1"}))
	assert.True(t, containsF1([]string{"a", " f1 "}))
	assert.False(t, containsF1([]string{"f2", "escape"}))
	assert.False(t, containsF1(nil))
}

func TestApplyInlineHelpMarkdown(t *testing.T) {
	result := applyInlineHelpMarkdown("**bold** and *italic* and `code`", true)
	assert.Contains(t, result, "bold")
	assert.Contains(t, result, "italic")
	assert.Contains(t, result, "code")
}

func TestRenderHelpMarkdown(t *testing.T) {
	help := "# Title\n\nSome text\n\n- bullet one\n- bullet two\n\n---\n\nMore text"
	result := renderHelpMarkdown(help, 40, true)
	assert.Contains(t, result, "Title")
	assert.Contains(t, result, "bullet one")
}

func TestRenderHelpMarkdown_MinWidth(t *testing.T) {
	result := renderHelpMarkdown("hello", 0, true)
	assert.Contains(t, result, "hello")
}
