package ui

// DisplaySchema controls how the interactive TUI renders arrays of objects.
// When present, arrays matching the schema render as a scrollable card list
// (title + subtitle + badges) instead of the default KEY/VALUE table, and
// drilling into an item shows a sectioned detail view.
type DisplaySchema struct {
	// Version identifies the schema format. Use "v1".
	Version string `json:"displaySchema,omitempty"`

	// Icon is an optional emoji/symbol shown before the collection title.
	Icon string `json:"icon,omitempty"`

	// CollectionTitle is an optional heading shown above the list view
	// (e.g., "Providers", "Services").
	CollectionTitle string `json:"collectionTitle,omitempty"`

	// List configures how an array of objects is rendered as a card list.
	// When nil, the default KEY/VALUE table is used.
	List *ListDisplayConfig `json:"list,omitempty"`

	// Detail configures how a single object is rendered when drilling in
	// from the list view. When nil, the default KEY/VALUE table is used.
	Detail *DetailDisplayConfig `json:"detail,omitempty"`
}

// ListDisplayConfig controls the card-list rendering for arrays of objects.
type ListDisplayConfig struct {
	// TitleField is the object key whose value is shown as the card title (bold).
	// Required for list view activation.
	TitleField string `json:"titleField"`

	// SubtitleField is the object key whose value is shown below the title (dimmed, truncated).
	SubtitleField string `json:"subtitleField,omitempty"`

	// SubtitleMaxLines limits how many lines the subtitle occupies (default: 1).
	SubtitleMaxLines int `json:"subtitleMaxLines,omitempty"`

	// BadgeFields lists object keys whose values are rendered as inline pills/tags
	// next to or below the title. Array values are expanded into individual badges.
	BadgeFields []string `json:"badgeFields,omitempty"`

	// SecondaryFields lists object keys shown as small metadata below the subtitle.
	SecondaryFields []string `json:"secondaryFields,omitempty"`

	// ArrayStyle overrides the global --array-style for this list.
	// Valid values: "index", "numbered", "bullet", "none".
	ArrayStyle string `json:"arrayStyle,omitempty"`
}

// DetailDisplayConfig controls how a single object is rendered in detail view.
type DetailDisplayConfig struct {
	// TitleField is the object key whose value is shown as the detail header.
	TitleField string `json:"titleField"`

	// Sections defines ordered groups of fields with specific layouts.
	// Fields not mentioned in any section are collected into a trailing
	// "Other" section using the default table layout.
	Sections []DetailSection `json:"sections,omitempty"`

	// HiddenFields lists object keys that are excluded from the detail view entirely.
	HiddenFields []string `json:"hiddenFields,omitempty"`
}

// DetailSection defines a group of fields rendered together with a specific layout.
type DetailSection struct {
	// Title is an optional heading shown above this section.
	Title string `json:"title,omitempty"`

	// Fields lists the object keys included in this section.
	Fields []string `json:"fields"`

	// Layout controls how the fields are rendered:
	//   - "inline"    — key: value pairs on a single line (e.g., "v1.0.0 · data")
	//   - "paragraph" — full-width wrapped text block
	//   - "tags"      — colored pill badges (for arrays of strings)
	//   - "table"     — standard KEY/VALUE table rows (default)
	Layout string `json:"layout,omitempty"`
}

// Layout constants for DetailSection.
const (
	DisplayLayoutInline    = "inline"
	DisplayLayoutParagraph = "paragraph"
	DisplayLayoutTags      = "tags"
	DisplayLayoutTable     = "table"
)

// ListPanelMode constants control the behaviour of the search panel in list view.
const (
	ListPanelModeSearch = "search" // Deep search across all fields, committed on Enter
	ListPanelModeFilter = "filter" // Real-time title+subtitle filter
)
