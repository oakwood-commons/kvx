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

	// Status configures a waiting/status screen view for async workflows
	// (e.g., device-code authentication, deployment progress). When set,
	// the data is rendered as a styled status panel with configurable
	// actions, an optional spinner, and timeout-based auto-close.
	Status *StatusDisplayConfig `json:"status,omitempty"`
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

// StatusDisplayConfig controls how data is rendered as an interactive status/waiting screen.
type StatusDisplayConfig struct {
	// TitleField is the data field whose value is displayed as the heading.
	TitleField string `json:"titleField"`

	// MessageField is the data field containing informational messages.
	// The value can be a single string or an array of strings.
	MessageField string `json:"messageField,omitempty"`

	// WaitMessage is the text shown next to the spinner while waiting
	// (e.g., "Waiting for authentication...").
	WaitMessage string `json:"waitMessage,omitempty"`

	// SuccessMessage is shown when the operation completes successfully.
	SuccessMessage string `json:"successMessage,omitempty"`

	// Timeout is a duration string (e.g., "30s", "2m") after which the
	// screen transitions to success and auto-exits. Ignored when a
	// programmatic Done channel is provided via Config.Done.
	Timeout string `json:"timeout,omitempty"`

	// DisplayFields lists data fields to show as labeled values on the status screen
	// (e.g., a device code or URL the user needs to copy/visit).
	DisplayFields []StatusFieldDisplay `json:"displayFields,omitempty"`

	// Actions defines interactive hotkey actions available on the status screen.
	Actions []StatusActionConfig `json:"actions,omitempty"`

	// DoneBehavior controls what happens after the operation completes.
	// Valid values: "exit-after-delay" (default), "wait-for-key".
	DoneBehavior string `json:"doneBehavior,omitempty"`

	// DoneDelay is the delay before auto-exit after completion (default: "2s").
	// Only used when DoneBehavior is "exit-after-delay" or empty.
	DoneDelay string `json:"doneDelay,omitempty"`
}

// StatusFieldDisplay defines a data field to display as a labeled value on the status screen.
type StatusFieldDisplay struct {
	// Label is the display label (e.g., "URL", "Code").
	Label string `json:"label"`

	// Field is the data field name whose value is shown.
	Field string `json:"field"`
}

// StatusActionConfig defines an interactive action on the status screen.
type StatusActionConfig struct {
	// Label is the display text shown in the action bar (e.g., "Copy code").
	Label string `json:"label"`

	// Type is the built-in action handler. Supported types:
	//   - "copy-value" — copies the value of Field to the system clipboard
	//   - "open-url"   — opens the value of Field in the default browser
	Type string `json:"type"`

	// Field is the data field whose value is acted upon.
	Field string `json:"field"`

	// Keys defines the hotkey per keybinding mode (vim, emacs, function).
	Keys StatusKeyBindings `json:"keys"`
}

// StatusKeyBindings defines per-mode key bindings for a status action.
type StatusKeyBindings struct {
	Vim      string `json:"vim,omitempty"`
	Emacs    string `json:"emacs,omitempty"`
	Function string `json:"function,omitempty"`
}

// StatusResult carries the outcome of an async operation for the status screen.
// Library consumers send this on the Config.Done channel when the operation finishes.
type StatusResult struct {
	// Err is non-nil if the operation failed.
	Err error
	// Message is an optional human-readable result message.
	Message string
}

// DoneBehavior constants for StatusDisplayConfig.
const (
	DoneBehaviorExitAfterDelay = "exit-after-delay"
	DoneBehaviorWaitForKey     = "wait-for-key"
)
