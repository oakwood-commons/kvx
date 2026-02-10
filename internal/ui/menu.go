package ui

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"text/template"

	tea "charm.land/bubbletea/v2"
)

// MenuItem represents a single menu entry (function key).
type MenuItem struct {
	Label    string
	Action   string
	Enabled  bool
	Popup    InfoPopupConfig
	HelpText string
	// Key bindings for this action in each mode
	Keys MenuKeyBindings
}

// MenuConfig defines labels and enabled state for function-key menu items.
type MenuConfig struct {
	F1  MenuItem
	F2  MenuItem
	F3  MenuItem
	F4  MenuItem
	F5  MenuItem
	F6  MenuItem
	F7  MenuItem
	F8  MenuItem
	F9  MenuItem
	F10 MenuItem
	F11 MenuItem
	F12 MenuItem
	// Action-based items (new structure)
	Items map[string]MenuItem
}

// KeyActionMap maps keys to actions for a specific mode.
type KeyActionMap map[string]string

var (
	defaultMenuOnce         sync.Once
	defaultMenuConfig       MenuConfig
	defaultMenuBuilding     int32
	defaultMenuTemplateData map[string]any
	currentMenuOnce         sync.Once
	currentMenuConfig       MenuConfig
	currentMenuActions      = defaultMenuActions()
	// Dynamic key-to-action mappings per mode, built from config
	keyActionMaps = make(map[KeyMode]KeyActionMap)
)

// DefaultMenuConfig returns the menu sourced from the embedded default configuration.
// Falls back to the legacy hard-coded menu only if the embedded config cannot be read.
func DefaultMenuConfig() MenuConfig {
	// Protect against re-entrancy before acquiring sync.Once; a recursive call would
	// otherwise deadlock while the Once lock is held. If we detect in-progress
	// initialization, fall back to the legacy hard-coded menu.
	if !atomic.CompareAndSwapInt32(&defaultMenuBuilding, 0, 1) {
		return fallbackDefaultMenuConfig()
	}
	defer atomic.StoreInt32(&defaultMenuBuilding, 0)

	defaultMenuOnce.Do(func() {
		cfg, err := EmbeddedDefaultConfig()
		if err != nil {
			defaultMenuConfig = fallbackDefaultMenuConfig()
			return
		}
		allowEdit := true
		if cfg.Features.AllowEditInput != nil {
			allowEdit = *cfg.Features.AllowEditInput
		}
		menu := MenuFromConfig(cfg.Menu, &allowEdit)
		helpText := GenerateHelpText(menu, allowEdit, nil, DefaultKeyMode)
		defaultMenuTemplateData = map[string]any{
			"config": map[string]any{
				"app": map[string]any{
					"about_line": fmt.Sprintf("%s: %s", cfg.About.Name, strings.TrimPrefix(cfg.About.Description, "A ")),
					"about": map[string]any{
						"name":        cfg.About.Name,
						"version":     cfg.About.Version,
						"description": cfg.About.Description,
						"license":     cfg.About.License,
						"author":      cfg.About.Author,
					},
					"help": map[string]any{
						"text": helpText,
					},
				},
			},
		}
		renderMenuTemplates(&menu, defaultMenuTemplateData)
		defaultMenuConfig = menu
	})

	if defaultMenuTemplateData != nil {
		// Re-render on each call to handle previously cached templates from earlier runs.
		renderMenuTemplates(&defaultMenuConfig, defaultMenuTemplateData)
	}

	return defaultMenuConfig
}

func fallbackDefaultMenuConfig() MenuConfig {
	// Default help popup: enabled with a friendly placeholder; config may override.
	helpEnabled := true
	helpModal := true
	helpItem := MenuItem{
		Label:    "help",
		Action:   "help",
		Enabled:  true,
		HelpText: "Toggle inline help",
		Keys:     MenuKeyBindings{Function: "f1", Vim: "?", Emacs: "f1"},
		Popup: InfoPopupConfig{
			Enabled: &helpEnabled,
			Text:    "kvx: terminal-based UI for exploring structured data in an interactive, navigable way.",
			Anchor:  "top",
			Justify: "center",
			Modal:   &helpModal,
		},
	}
	searchItem := MenuItem{Label: "search", Action: "search", Enabled: true, HelpText: "Start search", Keys: MenuKeyBindings{Function: "f3", Vim: "/", Emacs: "ctrl+s"}}
	filterItem := MenuItem{Label: "filter", Action: "filter", Enabled: true, HelpText: "Filter map keys", Keys: MenuKeyBindings{Function: "f4", Vim: "f", Emacs: "ctrl+l"}}
	copyItem := MenuItem{Label: "copy", Action: "copy", Enabled: true, HelpText: "Copy current expression/path", Keys: MenuKeyBindings{Function: "f5", Vim: "y", Emacs: "alt+w"}}
	exprItem := MenuItem{Label: "expr", Action: "expr_toggle", Enabled: true, HelpText: "Toggle expression input", Keys: MenuKeyBindings{Function: "f6", Vim: ":", Emacs: "alt+x"}}
	quitItem := MenuItem{Label: "quit", Action: "quit", Enabled: true, HelpText: "Quit", Keys: MenuKeyBindings{Function: "f10", Vim: "q", Emacs: "ctrl+q"}}

	menu := MenuConfig{
		F1:  helpItem,
		F2:  MenuItem{},
		F3:  searchItem,
		F4:  filterItem,
		F5:  copyItem,
		F6:  exprItem,
		F7:  MenuItem{},
		F8:  MenuItem{},
		F9:  MenuItem{},
		F10: quitItem,
		F11: MenuItem{},
		F12: MenuItem{},
		Items: map[string]MenuItem{
			"help":   helpItem,
			"search": searchItem,
			"filter": filterItem,
			"copy":   copyItem,
			"expr":   exprItem,
			"quit":   quitItem,
		},
	}
	// Build key-action maps for fallback config
	buildKeyActionMaps(menu)
	return menu
}

// SetMenuConfig overrides the current menu configuration.
func SetMenuConfig(cfg MenuConfig) {
	currentMenuConfig = cfg
	// Update vim/emacs keybindings from the new config
	UpdateKeyBindingsFromConfig(cfg)
}

// CurrentMenuConfig returns the active menu configuration.
func CurrentMenuConfig() MenuConfig {
	currentMenuOnce.Do(func() {
		currentMenuConfig = DefaultMenuConfig()
	})
	return currentMenuConfig
}

// MenuAction executes a menu action and may return a Bubble Tea command.
type MenuAction func(*Model) tea.Cmd

// defaultMenuActions returns the built-in menu action handlers.
func defaultMenuActions() map[string]MenuAction {
	return map[string]MenuAction{
		"help":        menuActionHelp,
		"expr_toggle": menuActionExprToggle,
		"search":      menuActionSearch,
		"filter":      menuActionFilter,
		"copy":        menuActionCopy,
		"quit":        menuActionQuit,
		"custom":      menuActionCustom,
		"noop":        func(_ *Model) tea.Cmd { return nil },
		"":            func(_ *Model) tea.Cmd { return nil },
	}
}

// SetMenuActions overrides all menu actions.
func SetMenuActions(actions map[string]MenuAction) {
	if actions == nil {
		currentMenuActions = defaultMenuActions()
		return
	}
	currentMenuActions = actions
}

// RegisterMenuAction registers or replaces a single menu action handler.
func RegisterMenuAction(name string, action MenuAction) {
	if currentMenuActions == nil {
		currentMenuActions = defaultMenuActions()
	}
	currentMenuActions[name] = action
}

// CurrentMenuActions returns the active menu actions map.
func CurrentMenuActions() map[string]MenuAction {
	if currentMenuActions == nil {
		currentMenuActions = defaultMenuActions()
	}
	return currentMenuActions
}

// MenuFromConfig builds a MenuConfig from YAML config plus an optional allowEditInput flag.
// Supports both action-based (new) and F-key based (legacy) config formats.
func MenuFromConfig(cfg MenuConfigYAML, allowEditInput *bool) MenuConfig {
	menu := fallbackDefaultMenuConfig()
	menu.Items = make(map[string]MenuItem)

	// Helper to copy from config to MenuItem
	set := func(src MenuItemConfig, dst *MenuItem, actionName string) {
		if src.Label != "" {
			dst.Label = src.Label
		}
		if src.HelpText != "" {
			dst.HelpText = src.HelpText
		}
		// Use explicit action if set, otherwise use the key name
		if src.Action != "" {
			dst.Action = src.Action
		} else if actionName != "" {
			dst.Action = actionName
		}
		if src.Enabled != nil {
			dst.Enabled = *src.Enabled
		}
		dst.Keys = src.Keys
		if InfoPopupHasData(src.Popup) {
			dst.Popup = src.Popup
		} else if src.PopupText != "" {
			dst.Popup = InfoPopupConfig{Text: src.PopupText}
		}
	}

	// Map function key strings (f1-f12) to MenuConfig fields
	fkeyToMenuPtr := func(fkey string) *MenuItem {
		switch strings.ToLower(fkey) {
		case "f1":
			return &menu.F1
		case "f2":
			return &menu.F2
		case "f3":
			return &menu.F3
		case "f4":
			return &menu.F4
		case "f5":
			return &menu.F5
		case "f6":
			return &menu.F6
		case "f7":
			return &menu.F7
		case "f8":
			return &menu.F8
		case "f9":
			return &menu.F9
		case "f10":
			return &menu.F10
		case "f11":
			return &menu.F11
		case "f12":
			return &menu.F12
		default:
			return nil
		}
	}

	// Process action-based config (new format)
	actionItems := []struct {
		name string
		cfg  MenuItemConfig
	}{
		{"help", cfg.Help},
		{"search", cfg.Search},
		{"filter", cfg.Filter},
		{"copy", cfg.Copy},
		{"expr", cfg.Expr},
		{"quit", cfg.Quit},
		{"custom", cfg.Custom},
	}

	for _, item := range actionItems {
		if item.cfg.Label == "" && item.cfg.Enabled == nil {
			continue // Skip empty items
		}
		mi := MenuItem{
			Label:    item.cfg.Label,
			HelpText: item.cfg.HelpText,
			Enabled:  true,
			Keys:     item.cfg.Keys,
		}
		// Action defaults to item name, but can be overridden (e.g., expr -> expr_toggle)
		if item.cfg.Action != "" {
			mi.Action = item.cfg.Action
		} else {
			mi.Action = item.name
		}
		if item.cfg.Enabled != nil {
			mi.Enabled = *item.cfg.Enabled
		}
		if InfoPopupHasData(item.cfg.Popup) {
			mi.Popup = item.cfg.Popup
		} else if item.cfg.PopupText != "" {
			mi.Popup = InfoPopupConfig{Text: item.cfg.PopupText}
		}

		menu.Items[item.name] = mi

		// Also populate F-key slots based on function key binding
		if fkey := item.cfg.Keys.Function; fkey != "" {
			if fkeyPtr := fkeyToMenuPtr(fkey); fkeyPtr != nil {
				*fkeyPtr = mi
			}
		}
	}

	// Legacy support: process F1-F12 directly if present (overrides action-based)
	set(cfg.F1, &menu.F1, "")
	set(cfg.F2, &menu.F2, "")
	set(cfg.F3, &menu.F3, "")
	set(cfg.F4, &menu.F4, "")
	set(cfg.F5, &menu.F5, "")
	set(cfg.F6, &menu.F6, "")
	set(cfg.F7, &menu.F7, "")
	set(cfg.F8, &menu.F8, "")
	set(cfg.F9, &menu.F9, "")
	set(cfg.F10, &menu.F10, "")
	set(cfg.F11, &menu.F11, "")
	set(cfg.F12, &menu.F12, "")

	// Build key-to-action maps for each mode
	buildKeyActionMaps(menu)

	// If editing is disabled globally, disable any expr_toggle key.
	if allowEditInput != nil && !*allowEditInput {
		items := []*MenuItem{
			&menu.F1, &menu.F2, &menu.F3, &menu.F4, &menu.F5, &menu.F6,
			&menu.F7, &menu.F8, &menu.F9, &menu.F10, &menu.F11, &menu.F12,
		}
		for _, it := range items {
			if it.Action == "expr_toggle" {
				it.Enabled = false
			}
		}
		// Also disable in Items map
		for name, item := range menu.Items {
			if item.Action == "expr_toggle" {
				item.Enabled = false
				menu.Items[name] = item
			}
		}
	}
	return menu
}

// buildKeyActionMaps builds the key-to-action mappings for each mode from the menu config.
func buildKeyActionMaps(menu MenuConfig) {
	keyActionMaps[KeyModeFunction] = make(KeyActionMap)
	keyActionMaps[KeyModeVim] = make(KeyActionMap)
	keyActionMaps[KeyModeEmacs] = make(KeyActionMap)

	for name, item := range menu.Items {
		if !item.Enabled {
			continue
		}
		action := item.Action
		if action == "" {
			action = name
		}
		if item.Keys.Function != "" {
			keyActionMaps[KeyModeFunction][strings.ToLower(item.Keys.Function)] = action
		}
		if item.Keys.Vim != "" {
			keyActionMaps[KeyModeVim][item.Keys.Vim] = action
		}
		if item.Keys.Emacs != "" {
			keyActionMaps[KeyModeEmacs][item.Keys.Emacs] = action
		}
	}

	// Also update vim/emacs keybinding maps for key handling
	UpdateKeyBindingsFromConfig(menu)
}

// GetKeyActionMap returns the key-to-action map for a specific key mode.
func GetKeyActionMap(mode KeyMode) KeyActionMap {
	if m, ok := keyActionMaps[mode]; ok {
		return m
	}
	return nil
}

// GetActionForKey returns the action associated with a key in the given mode.
func GetActionForKey(mode KeyMode, key string) string {
	if m := GetKeyActionMap(mode); m != nil {
		return m[key]
	}
	return ""
}

// GetItemForAction returns the MenuItem for a given action name.
func GetItemForAction(actionName string) *MenuItem {
	menu := CurrentMenuConfig()
	if item, ok := menu.Items[actionName]; ok {
		return &item
	}
	return nil
}

func renderMenuTemplates(menu *MenuConfig, data map[string]interface{}) {
	render := func(val string) string {
		if !strings.Contains(val, "{{") {
			return val
		}
		tmpl, err := template.New("popup").Option("missingkey=zero").Parse(val)
		if err != nil {
			return val
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return val
		}
		return buf.String()
	}

	renderPopup := func(p *InfoPopupConfig) {
		if p == nil {
			return
		}
		if p.Title != "" {
			p.Title = render(p.Title)
		}
		if p.Text != "" {
			p.Text = render(p.Text)
		}
	}

	popups := []*InfoPopupConfig{
		&menu.F1.Popup, &menu.F2.Popup, &menu.F3.Popup, &menu.F4.Popup, &menu.F5.Popup, &menu.F6.Popup,
		&menu.F7.Popup, &menu.F8.Popup, &menu.F9.Popup, &menu.F10.Popup, &menu.F11.Popup, &menu.F12.Popup,
	}
	for _, p := range popups {
		renderPopup(p)
	}
}

// MenuItems returns ordered key/item pairs for F1-F12.
func MenuItems(cfg MenuConfig) []struct {
	Key  string
	Item MenuItem
} {
	return []struct {
		Key  string
		Item MenuItem
	}{
		{"F1", cfg.F1},
		{"F2", cfg.F2},
		{"F3", cfg.F3},
		{"F4", cfg.F4},
		{"F5", cfg.F5},
		{"F6", cfg.F6},
		{"F7", cfg.F7},
		{"F8", cfg.F8},
		{"F9", cfg.F9},
		{"F10", cfg.F10},
		{"F11", cfg.F11},
		{"F12", cfg.F12},
	}
}
