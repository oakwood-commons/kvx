package cmd

import ui "github.com/oakwood-commons/kvx/internal/ui"

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
