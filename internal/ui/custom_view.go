package ui

import tea "charm.land/bubbletea/v2"

// CustomView is the interface that every custom view mode (list, detail, status)
// must implement.  The panel-layout layer calls these methods instead of
// hard-coding per-mode logic, so adding a new view mode requires only:
//  1. A new type that implements CustomView.
//  2. A new case in updateViewMode() to instantiate it.
//  3. A new case in the key-routing switch in Model.Update().
//
// Everything else—title promotion, footer, flash messages, content rendering,
// row counts—flows through the interface automatically.
type CustomView interface {
	// Title returns the string shown in the top panel border.
	Title() string

	// Render returns the content for the data panel (inside the border).
	// width and height are the available dimensions after borders/status.
	Render(width, height int, noColor bool) string

	// RowCount returns the item/row count, the 1-based selected index,
	// and a footer label such as "list: 3/15" or "detail".
	RowCount() (count int, selected int, label string)

	// FooterBar returns an optional custom footer bar (key hints).
	// Return "" to use the default footer.
	FooterBar() string

	// FlashMessage returns an active flash message and whether it is an error.
	// Success messages (e.g. "✓ Copied") should return isError=false so they
	// render in the default info style; warning/error messages (e.g. "⚠ …")
	// should return isError=true to render in the error style.
	FlashMessage() (msg string, isError bool)

	// SearchTitle returns a title override for the search/filter input panel.
	// Return "" to keep the default "Search" title.
	SearchTitle() string

	// HandlesSearch returns true if this view handles search/filter internally
	// (bypassing the default deep-search pipeline).
	HandlesSearch() bool

	// Init returns any initial commands (e.g. spinner tick, timeout timer).
	// Called once when the view is first activated.
	Init() tea.Cmd

	// Update handles async messages that belong to this view (e.g.
	// spinner ticks, done signals, flash clear timers).  The view should
	// ignore messages it does not recognise and return (self, nil).
	Update(msg tea.Msg) (CustomView, tea.Cmd)
}
