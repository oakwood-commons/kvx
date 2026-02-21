package ui

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"
)

// copyToClipboardFn and openURLFn are the active implementations for clipboard
// and browser operations. Tests replace them with no-ops via
// stubPlatformActions() to prevent side effects.
var (
	copyToClipboardFn = copyToClipboardImpl
	openURLFn         = openURLImpl
)

// CopyToClipboard copies text to the system clipboard.
func CopyToClipboard(text string) error { return copyToClipboardFn(text) }

// OpenURL opens a URL in the default browser.
func OpenURL(url string) error { return openURLFn(url) }

// StubPlatformActions replaces clipboard and browser functions with no-ops
// and returns a restore function. Use in tests to prevent side effects.
func StubPlatformActions() (restore func()) {
	origCopy := copyToClipboardFn
	origOpen := openURLFn
	copyToClipboardFn = func(string) error { return nil }
	openURLFn = func(string) error { return nil }
	return func() {
		copyToClipboardFn = origCopy
		openURLFn = origOpen
	}
}

// copyToClipboardImpl is the real clipboard implementation.
func copyToClipboardImpl(text string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "pbcopy")
	case "linux":
		// Try xclip first, then xsel, then wl-copy (Wayland)
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.CommandContext(ctx, "xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.CommandContext(ctx, "xsel", "--clipboard", "--input")
		} else if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.CommandContext(ctx, "wl-copy")
		} else {
			return fmt.Errorf("no clipboard command found (install xclip, xsel, or wl-clipboard)")
		}
	case "windows":
		cmd = exec.CommandContext(ctx, "clip")
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	_, _ = stdin.Write([]byte(text))
	_ = stdin.Close()

	return cmd.Wait()
}

// openURLImpl is the real browser-open implementation.
// Uses a detached context since the child process outlives the caller.
func openURLImpl(url string) error {
	ctx := context.Background()

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "open", url)
	case "linux":
		if _, err := exec.LookPath("xdg-open"); err == nil {
			cmd = exec.CommandContext(ctx, "xdg-open", url)
		} else {
			return fmt.Errorf("xdg-open not found (install xdg-utils)")
		}
	case "windows":
		cmd = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}
