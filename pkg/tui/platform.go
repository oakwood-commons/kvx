package tui

import "github.com/oakwood-commons/kvx/internal/ui"

// CopyToClipboard copies text to the system clipboard using platform-specific
// commands (pbcopy on macOS, xclip/xsel/wl-copy on Linux, clip on Windows).
//
// This is useful in StatusAction callbacks:
//
//	tui.StatusActionConfig{
//	    Label: "Copy code",
//	    Type:  "copy-value",
//	    Field: "code",
//	}
//
// Or for custom callbacks in library consumers.
func CopyToClipboard(text string) error {
	return ui.CopyToClipboard(text)
}

// OpenURL opens the given URL in the default system browser using
// platform-specific commands (open on macOS, xdg-open on Linux,
// rundll32 on Windows).
//
// This is useful in StatusAction callbacks:
//
//	tui.StatusActionConfig{
//	    Label: "Open URL",
//	    Type:  "open-url",
//	    Field: "url",
//	}
//
// Or for custom callbacks in library consumers.
func OpenURL(url string) error {
	return ui.OpenURL(url)
}
