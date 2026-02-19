package completion

import (
	"strings"
)

// FormatFunctionOneLiner returns a single-line help string for the status bar.
// Shows signature and description only — use the palette (Ctrl+Space) for examples.
func FormatFunctionOneLiner(fn FunctionMetadata) string {
	sig := FormatFunctionSignature(fn)
	desc := strings.TrimSpace(fn.Description)

	if desc == "" {
		return sig
	}

	return sig + " — " + desc
}

// FormatFunctionSignature returns a display-ready signature for a function,
// falling back to name() if no signature is set.
func FormatFunctionSignature(fn FunctionMetadata) string {
	if fn.Signature != "" {
		return fn.Signature
	}
	return fn.Name + "()"
}

// FormatFunctionLines returns multi-line help text: signature, description,
// and up to maxExamples example lines. Each element is a separate line.
func FormatFunctionLines(fn FunctionMetadata, maxExamples int) []string {
	lines := make([]string, 0, 4)

	sig := FormatFunctionSignature(fn)
	lines = append(lines, sig)

	if fn.Description != "" {
		lines = append(lines, fn.Description)
	}

	for i, ex := range fn.Examples {
		if maxExamples > 0 && i >= maxExamples {
			break
		}
		ex = strings.TrimSpace(ex)
		if ex != "" {
			lines = append(lines, "  "+ex)
		}
	}

	return lines
}
