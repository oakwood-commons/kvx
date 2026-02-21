package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/oakwood-commons/kvx/pkg/tui"
)

func main() {
	keyMode := flag.String("key-mode", "vim", "keybinding mode: vim, emacs, or function")
	noTUI := flag.Bool("no-tui", false, "disable TUI, use plain text output")
	timeout := flag.Duration("timeout", 10*time.Second, "simulated auth timeout")
	flag.Parse()

	data := map[string]any{
		"title": "Sign in to Entra",
		"url":   "https://microsoft.com/devicelogin",
		"code":  "EH5HFPGJJ",
		"user":  "user@example.com",
		"messages": []any{
			"Already authenticated as user@example.com",
			"Use 'myapp auth logout entra' to sign out first",
		},
	}

	schema := &tui.DisplaySchema{
		Version: "v1",
		Status: &tui.StatusDisplayConfig{
			TitleField:     "title",
			MessageField:   "messages",
			WaitMessage:    "Waiting for authentication...",
			SuccessMessage: "Authenticated successfully!",
			DoneBehavior:   tui.DoneBehaviorExitAfterDelay,
			DoneDelay:      "2s",
			DisplayFields: []tui.StatusFieldDisplay{
				{Label: "URL", Field: "url"},
				{Label: "Code", Field: "code"},
			},
			Actions: []tui.StatusActionConfig{
				{
					Label: "Copy code",
					Type:  "copy-value",
					Field: "code",
					Keys:  tui.StatusKeyBindings{Vim: "c", Emacs: "alt+c", Function: "f2"},
				},
				{
					Label: "Open URL",
					Type:  "open-url",
					Field: "url",
					Keys:  tui.StatusKeyBindings{Vim: "o", Emacs: "alt+o", Function: "f3"},
				},
			},
		},
	}

	if *noTUI {
		fmt.Fprintln(os.Stdout, "Sign in to Entra")
		fmt.Fprintln(os.Stdout)
		fmt.Fprintln(os.Stdout, "  Already authenticated as user@example.com")
		fmt.Fprintln(os.Stdout, "  Use 'myapp auth logout entra' to sign out first")
		fmt.Fprintln(os.Stdout)
		fmt.Fprintf(os.Stdout, "  To sign in, use a web browser to open the page:\n")
		fmt.Fprintf(os.Stdout, "    %s\n", data["url"])
		fmt.Fprintf(os.Stdout, "  Enter the code: %s\n", data["code"])
		fmt.Fprintln(os.Stdout)
		fmt.Fprintln(os.Stdout, "  Waiting for authentication...")
		time.Sleep(*timeout)
		fmt.Fprintln(os.Stdout, "  Authenticated successfully!")
		return
	}

	done := make(chan tui.StatusResult, 1)
	go func() {
		time.Sleep(*timeout)
		done <- tui.StatusResult{Message: "Authenticated as user@example.com"}
	}()

	cfg := tui.DefaultConfig()
	cfg.AppName = "myapp"
	cfg.DisplaySchema = schema
	cfg.KeyMode = *keyMode
	cfg.Done = done

	if err := tui.Run(data, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
