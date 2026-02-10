package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

// ApplyStartupKeys simulates startup keypresses (Vim-like tokens and literal text).
// It mutates the provided model in place.
func ApplyStartupKeys(m *Model, keys []string) {
	if len(keys) == 0 || m == nil {
		return
	}
	sawEsc := false
	for _, raw := range keys {
		token := strings.TrimSpace(raw)
		if token == "" {
			continue
		}
		// Leading backslash forces literal text (e.g., "\\<f12>").
		if strings.HasPrefix(token, `\`) {
			token = strings.TrimPrefix(token, `\`)
			for _, r := range token {
				msg := tea.KeyPressMsg{Code: r, Text: string(r)}
				if updated, _ := m.Update(msg); updated != nil {
					if um, ok := updated.(*Model); ok {
						*m = *um
					}
				}
			}
			continue
		}

		// Parse token that may contain both vim-style keys and literal text (e.g., "<F1>rwo")
		// Split into vim-style tokens and literal text segments
		segments := parseTokenSegments(token)
		for _, segment := range segments {
			if segment.isVimKey {
				// Process as vim-style key
				if msgs, ok := keyMsgsFromToken(segment.text); ok {
					for _, msg := range msgs {
						if msg.Code == tea.KeyEscape {
							sawEsc = true
						}
						if updated, _ := m.Update(msg); updated != nil {
							if um, ok := updated.(*Model); ok {
								*m = *um
							}
						}
					}
				}
			} else {
				// Process as literal text
				for _, r := range segment.text {
					msg := tea.KeyPressMsg{Code: r, Text: string(r)}
					if updated, _ := m.Update(msg); updated != nil {
						if um, ok := updated.(*Model); ok {
							*m = *um
						}
					}
				}
			}
		}
	}

	// Ensure Esc in startup keys actually clears any visible popup unless permanent.
	if sawEsc && m.ShowInfoPopup && !m.InfoPopupPermanent {
		m.InfoPopupEnabled = true
		m.InfoPopupPermanent = false
		m.ShowInfoPopup = false
	}
}

// tokenSegment represents a parsed segment of a token (either a vim-style key or literal text)
type tokenSegment struct {
	text     string
	isVimKey bool
}

// parseTokenSegments splits a token into segments of vim-style keys and literal text.
// Example: "<F1>rwo" -> [segment{text: "<F1>", isVimKey: true}, segment{text: "rwo", isVimKey: false}]
func parseTokenSegments(token string) []tokenSegment {
	var segments []tokenSegment
	remaining := token

	for len(remaining) > 0 {
		// Look for vim-style key pattern: <...>
		startIdx := strings.Index(remaining, "<")
		if startIdx == -1 {
			// No more vim-style keys, rest is literal text
			if len(remaining) > 0 {
				segments = append(segments, tokenSegment{text: remaining, isVimKey: false})
			}
			break
		}

		// Add any literal text before the vim-style key
		if startIdx > 0 {
			segments = append(segments, tokenSegment{text: remaining[:startIdx], isVimKey: false})
		}

		// Find the closing >
		endIdx := strings.Index(remaining[startIdx:], ">")
		if endIdx == -1 {
			// No closing >, treat rest as literal text
			segments = append(segments, tokenSegment{text: remaining, isVimKey: false})
			break
		}

		// Extract the vim-style key (including < and >)
		vimKey := remaining[startIdx : startIdx+endIdx+1]
		segments = append(segments, tokenSegment{text: vimKey, isVimKey: true})

		// Continue with the rest of the token
		remaining = remaining[startIdx+endIdx+1:]
	}

	return segments
}

// keyMsgsFromToken parses a Vim-like token into key messages.
// Examples: "<Esc>", "<CR>", "<Tab>", "<Space>", "<BS>", "<C-c>", "<C-[>", "<F3>".
// Only <...> forms are treated as keys; everything else is literal text.
func keyMsgsFromToken(token string) ([]tea.KeyPressMsg, bool) {
	if token == "" {
		return nil, false
	}
	if strings.HasPrefix(token, "<") && strings.HasSuffix(token, ">") {
		inner := strings.TrimSuffix(strings.TrimPrefix(token, "<"), ">")
		lower := strings.ToLower(inner)
		switch lower {
		case "esc", "c-[", "escape":
			return []tea.KeyPressMsg{{Code: tea.KeyEscape}}, true
		case "cr", "enter", "return":
			return []tea.KeyPressMsg{{Code: tea.KeyEnter}}, true
		case "tab":
			return []tea.KeyPressMsg{{Code: tea.KeyTab}}, true
		case "space":
			return []tea.KeyPressMsg{{Code: ' ', Text: " "}}, true
		case "bs", "backspace":
			return []tea.KeyPressMsg{{Code: tea.KeyBackspace}}, true
		case "left":
			return []tea.KeyPressMsg{{Code: tea.KeyLeft}}, true
		case "right":
			return []tea.KeyPressMsg{{Code: tea.KeyRight}}, true
		case "up":
			return []tea.KeyPressMsg{{Code: tea.KeyUp}}, true
		case "down":
			return []tea.KeyPressMsg{{Code: tea.KeyDown}}, true
		case "home":
			return []tea.KeyPressMsg{{Code: tea.KeyHome}}, true
		case "end":
			return []tea.KeyPressMsg{{Code: tea.KeyEnd}}, true
		case "c-c":
			return []tea.KeyPressMsg{{Code: 0x03}}, true // Ctrl+C
		case "c-d":
			return []tea.KeyPressMsg{{Code: 0x04}}, true // Ctrl+D
		}
		if strings.HasPrefix(lower, "f") {
			num := strings.TrimPrefix(lower, "f")
			switch num {
			case "1":
				return []tea.KeyPressMsg{{Code: tea.KeyF1}}, true
			case "2":
				return []tea.KeyPressMsg{{Code: tea.KeyF2}}, true
			case "3":
				return []tea.KeyPressMsg{{Code: tea.KeyF3}}, true
			case "4":
				return []tea.KeyPressMsg{{Code: tea.KeyF4}}, true
			case "5":
				return []tea.KeyPressMsg{{Code: tea.KeyF5}}, true
			case "6":
				return []tea.KeyPressMsg{{Code: tea.KeyF6}}, true
			case "7":
				return []tea.KeyPressMsg{{Code: tea.KeyF7}}, true
			case "8":
				return []tea.KeyPressMsg{{Code: tea.KeyF8}}, true
			case "9":
				return []tea.KeyPressMsg{{Code: tea.KeyF9}}, true
			case "10":
				return []tea.KeyPressMsg{{Code: tea.KeyF10}}, true
			case "11":
				return []tea.KeyPressMsg{{Code: tea.KeyF11}}, true
			case "12":
				return []tea.KeyPressMsg{{Code: tea.KeyF12}}, true
			}
		}
		return nil, false
	}
	return nil, false
}
