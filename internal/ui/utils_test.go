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
