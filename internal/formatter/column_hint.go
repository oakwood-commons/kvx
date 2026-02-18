package formatter

// ColumnHint provides display hints for a specific column in columnar table rendering.
// This is the internal (formatter-level) representation; the public API exposes
// [tui.ColumnHint] which maps to this type.
type ColumnHint struct {
	// MaxWidth caps the column width (in characters). 0 = no cap.
	MaxWidth int

	// Priority controls column importance when shrinking.
	// Higher values resist shrinking; lower values shrink first.
	Priority int

	// Align controls text alignment: "right" or "left" (default).
	Align string

	// DisplayName overrides the column header text.
	// Empty string means use the original field name.
	DisplayName string
}
