package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
	ui "github.com/oakwood-commons/kvx/internal/ui"
)

// renderPlainTextStatus renders the status screen data as formatted plain text
// for non-interactive CLI output. Returns the rendered text and true if a status
// schema was applied, or ("", false) if no status schema is present.
func renderPlainTextStatus(data any, ds *ui.DisplaySchema) (string, bool) {
	if ds == nil || ds.Status == nil {
		return "", false
	}
	sc := ds.Status
	dataMap, ok := data.(map[string]any)
	if !ok {
		return "", false
	}

	var lines []string

	// Title
	if sc.TitleField != "" {
		if val, ok := dataMap[sc.TitleField]; ok {
			lines = append(lines, fmt.Sprintf("%v", val))
			lines = append(lines, "")
		}
	}

	// Messages
	if sc.MessageField != "" {
		if val, ok := dataMap[sc.MessageField]; ok {
			switch v := val.(type) {
			case string:
				lines = append(lines, "  "+v)
			case []any:
				for _, item := range v {
					lines = append(lines, fmt.Sprintf("  %v", item))
				}
			default:
				lines = append(lines, fmt.Sprintf("  %v", v))
			}
			lines = append(lines, "")
		}
	}

	// Display fields — labeled key-value pairs
	for _, df := range sc.DisplayFields {
		if df.Field == "" {
			continue
		}
		val, ok := dataMap[df.Field]
		if !ok {
			continue
		}
		v := fmt.Sprintf("%v", val)
		if isURL(v) {
			v = ansi.SetHyperlink(v) + v + ansi.ResetHyperlink()
		}
		lines = append(lines, fmt.Sprintf("  %s: %s", df.Label, v))
	}
	if len(sc.DisplayFields) > 0 {
		lines = append(lines, "")
	}

	// Wait message (if configured)
	if sc.WaitMessage != "" {
		lines = append(lines, "  "+sc.WaitMessage)
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n"), true
}

// isURL returns true if the value looks like an HTTP or HTTPS URL.
func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// renderSnapshotOutput centralizes snapshot sizing, help loading, and model configuration.
func renderSnapshotOutput(cfg ui.ThemeConfigFile, renderRoot interface{}, root interface{}, appName string, startKeys []string, expr string, noColor bool, widthFlag, heightFlag int, detectedW, detectedH int, configPath string, debugLog bool, dc *debugCollector, debugLabel string, keyMode ui.KeyMode) string {
	sizing := resolveSnapshotSize(widthFlag, heightFlag, detectedW, detectedH)
	if debugLog && dc != nil {
		if debugLabel != "" {
			dc.Printf("DBG: Snapshot size resolved (%s): width=%d height=%d (flag width=%d height=%d, detected width=%d height=%d)\n", debugLabel, sizing.Width, sizing.Height, widthFlag, heightFlag, sizing.DetectedWidth, sizing.DetectedHeight)
		} else {
			dc.Printf("DBG: Snapshot size resolved: width=%d height=%d (flag width=%d height=%d, detected width=%d height=%d)\n", sizing.Width, sizing.Height, widthFlag, heightFlag, sizing.DetectedWidth, sizing.DetectedHeight)
		}
	}

	helpTitle, helpText := loadHelp(configPath, keyMode)
	return renderSnapshotView(renderRoot, root, appName, helpTitle, helpText, startKeys, expr, noColor, sizing, func(m *ui.Model) {
		applySnapshotConfigToModel(m, cfg)
		if parsedDisplaySchema != nil {
			m.DisplaySchema = parsedDisplaySchema
		}
	})
}
