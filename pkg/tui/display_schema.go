package tui

import (
	"encoding/json"
	"fmt"

	"github.com/oakwood-commons/kvx/internal/ui"
)

// DisplaySchema controls how the interactive TUI renders arrays of objects.
// When present, arrays matching the schema render as a scrollable card list
// (title + subtitle + badges) instead of the default KEY/VALUE table, and
// drilling into an item shows a sectioned detail view.
//
// A DisplaySchema can be:
//   - constructed programmatically in Go code
//   - parsed from a standalone JSON document via [ParseDisplaySchema]
//   - extracted from JSON Schema vendor extensions (x-kvx-*) via [ParseSchemaWithDisplay]
type DisplaySchema = ui.DisplaySchema

// ListDisplayConfig controls the card-list rendering for arrays of objects.
type ListDisplayConfig = ui.ListDisplayConfig

// DetailDisplayConfig controls how a single object is rendered in detail view.
type DetailDisplayConfig = ui.DetailDisplayConfig

// DetailSection defines a group of fields rendered together with a specific layout.
type DetailSection = ui.DetailSection

// StatusDisplayConfig controls how data is rendered as an interactive status/waiting screen.
type StatusDisplayConfig = ui.StatusDisplayConfig

// StatusFieldDisplay defines a data field to display as a labeled value on the status screen.
type StatusFieldDisplay = ui.StatusFieldDisplay

// StatusActionConfig defines an interactive action on the status screen.
type StatusActionConfig = ui.StatusActionConfig

// StatusKeyBindings defines per-mode key bindings for a status action.
type StatusKeyBindings = ui.StatusKeyBindings

// StatusResult carries the outcome of an async operation for the status screen.
type StatusResult = ui.StatusResult

// DoneBehavior constants for StatusDisplayConfig.
const (
	DoneBehaviorExitAfterDelay = ui.DoneBehaviorExitAfterDelay
	DoneBehaviorWaitForKey     = ui.DoneBehaviorWaitForKey
)

// Layout constants for DetailSection.
const (
	DisplaySchemaLayoutInline    = ui.DisplayLayoutInline
	DisplaySchemaLayoutParagraph = ui.DisplayLayoutParagraph
	DisplaySchemaLayoutTags      = ui.DisplayLayoutTags
	DisplaySchemaLayoutTable     = ui.DisplayLayoutTable
)

// ParseDisplaySchema parses a standalone display schema JSON document.
// The document is identified by the presence of a "displaySchema" key.
//
// Example:
//
//	{
//	  "displaySchema": "v1",
//	  "icon": "📦",
//	  "collectionTitle": "Providers",
//	  "list": {
//	    "titleField": "name",
//	    "subtitleField": "description"
//	  }
//	}
func ParseDisplaySchema(data []byte) (*DisplaySchema, error) {
	// Quick check: is this a display schema doc?
	var probe map[string]any
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("invalid display schema JSON: %w", err)
	}
	if _, ok := probe["displaySchema"]; !ok {
		return nil, fmt.Errorf("missing \"displaySchema\" key: not a display schema document")
	}

	var ds DisplaySchema
	if err := json.Unmarshal(data, &ds); err != nil {
		return nil, fmt.Errorf("invalid display schema: %w", err)
	}

	// Parse x-kvx-status from the raw map (not in the base struct JSON tags)
	if statusRaw, ok := probe["x-kvx-status"]; ok {
		statusBytes, err := json.Marshal(statusRaw)
		if err == nil {
			var sc StatusDisplayConfig
			if err := json.Unmarshal(statusBytes, &sc); err == nil {
				ds.Status = &sc
			}
		}
	}

	if err := validateDisplaySchema(&ds); err != nil {
		return nil, err
	}
	return &ds, nil
}

// extractDisplaySchemaFromJSONSchema extracts x-kvx-* vendor extensions from
// an already-unmarshalled JSON Schema map. Returns nil when no extensions are present.
func extractDisplaySchemaFromJSONSchema(raw map[string]any) *DisplaySchema {
	// Find the right level: for type=array, look at the top level for x-kvx-* keys
	target := raw
	if typ, _ := raw["type"].(string); typ == "array" {
		// x-kvx-* keys live on the array schema itself
		target = raw
	}

	// Check for any x-kvx-* key
	hasExtension := false
	for k := range target {
		if len(k) > 5 && k[:5] == "x-kvx" {
			hasExtension = true
			break
		}
	}
	if !hasExtension {
		return nil
	}

	ds := &DisplaySchema{Version: "v1"}

	if icon, ok := target["x-kvx-icon"].(string); ok {
		ds.Icon = icon
	}
	if title, ok := target["x-kvx-collectionTitle"].(string); ok {
		ds.CollectionTitle = title
	}

	// x-kvx-list
	if listRaw, ok := target["x-kvx-list"].(map[string]any); ok {
		list := &ListDisplayConfig{}
		if v, ok := listRaw["titleField"].(string); ok {
			list.TitleField = v
		}
		if v, ok := listRaw["subtitleField"].(string); ok {
			list.SubtitleField = v
		}
		if v, ok := listRaw["subtitleMaxLines"]; ok {
			if n, ok := toInt(v); ok {
				list.SubtitleMaxLines = n
			}
		}
		if v, ok := listRaw["badgeFields"]; ok {
			list.BadgeFields = extractStringArray(v)
		}
		if v, ok := listRaw["secondaryFields"]; ok {
			list.SecondaryFields = extractStringArray(v)
		}
		if v, ok := listRaw["arrayStyle"].(string); ok {
			list.ArrayStyle = v
		}
		ds.List = list
	}

	// x-kvx-status
	if statusRaw, ok := target["x-kvx-status"].(map[string]any); ok {
		ds.Status = parseStatusExtension(statusRaw)
	}

	// x-kvx-detail
	if detailRaw, ok := target["x-kvx-detail"].(map[string]any); ok {
		detail := &DetailDisplayConfig{}
		if v, ok := detailRaw["titleField"].(string); ok {
			detail.TitleField = v
		}
		if v, ok := detailRaw["hiddenFields"]; ok {
			detail.HiddenFields = extractStringArray(v)
		}
		if sectionsRaw, ok := detailRaw["sections"].([]any); ok {
			for _, sRaw := range sectionsRaw {
				sMap, ok := sRaw.(map[string]any)
				if !ok {
					continue
				}
				section := DetailSection{}
				if v, ok := sMap["title"].(string); ok {
					section.Title = v
				}
				if v, ok := sMap["fields"]; ok {
					section.Fields = extractStringArray(v)
				}
				if v, ok := sMap["layout"].(string); ok {
					section.Layout = v
				}
				detail.Sections = append(detail.Sections, section)
			}
		}
		ds.Detail = detail
	}

	return ds
}

// parseStatusExtension parses a raw x-kvx-status map into a StatusDisplayConfig.
func parseStatusExtension(raw map[string]any) *StatusDisplayConfig {
	sc := &StatusDisplayConfig{}
	if v, ok := raw["titleField"].(string); ok {
		sc.TitleField = v
	}
	if v, ok := raw["messageField"].(string); ok {
		sc.MessageField = v
	}
	if v, ok := raw["waitMessage"].(string); ok {
		sc.WaitMessage = v
	}
	if v, ok := raw["successMessage"].(string); ok {
		sc.SuccessMessage = v
	}
	if v, ok := raw["timeout"].(string); ok {
		sc.Timeout = v
	}
	if v, ok := raw["doneBehavior"].(string); ok {
		sc.DoneBehavior = v
	}
	if v, ok := raw["doneDelay"].(string); ok {
		sc.DoneDelay = v
	}
	if dfRaw, ok := raw["displayFields"].([]any); ok {
		for _, dRaw := range dfRaw {
			dMap, ok := dRaw.(map[string]any)
			if !ok {
				continue
			}
			df := StatusFieldDisplay{}
			if v, ok := dMap["label"].(string); ok {
				df.Label = v
			}
			if v, ok := dMap["field"].(string); ok {
				df.Field = v
			}
			sc.DisplayFields = append(sc.DisplayFields, df)
		}
	}
	if actionsRaw, ok := raw["actions"].([]any); ok {
		for _, aRaw := range actionsRaw {
			aMap, ok := aRaw.(map[string]any)
			if !ok {
				continue
			}
			action := StatusActionConfig{}
			if v, ok := aMap["label"].(string); ok {
				action.Label = v
			}
			if v, ok := aMap["type"].(string); ok {
				action.Type = v
			}
			if v, ok := aMap["field"].(string); ok {
				action.Field = v
			}
			if keysRaw, ok := aMap["keys"].(map[string]any); ok {
				if v, ok := keysRaw["vim"].(string); ok {
					action.Keys.Vim = v
				}
				if v, ok := keysRaw["emacs"].(string); ok {
					action.Keys.Emacs = v
				}
				if v, ok := keysRaw["function"].(string); ok {
					action.Keys.Function = v
				}
			}
			sc.Actions = append(sc.Actions, action)
		}
	}
	return sc
}

// validateDisplaySchema checks that a display schema has the minimum required fields.
func validateDisplaySchema(ds *DisplaySchema) error {
	if ds.List != nil && ds.List.TitleField == "" {
		return fmt.Errorf("display schema: list.titleField is required")
	}
	if ds.Detail != nil {
		for i, s := range ds.Detail.Sections {
			if len(s.Fields) == 0 {
				return fmt.Errorf("display schema: detail.sections[%d].fields is required", i)
			}
			switch s.Layout {
			case "", DisplaySchemaLayoutInline, DisplaySchemaLayoutParagraph,
				DisplaySchemaLayoutTags, DisplaySchemaLayoutTable:
				// valid
			default:
				return fmt.Errorf("display schema: detail.sections[%d].layout: unknown layout %q", i, s.Layout)
			}
		}
	}
	if ds.Status != nil {
		if ds.Status.TitleField == "" {
			return fmt.Errorf("display schema: status.titleField is required")
		}
		for i, a := range ds.Status.Actions {
			if a.Label == "" {
				return fmt.Errorf("display schema: status.actions[%d].label is required", i)
			}
			switch a.Type {
			case "copy-value", "open-url":
				// valid
			default:
				return fmt.Errorf("display schema: status.actions[%d].type: unknown type %q", i, a.Type)
			}
		}
	}
	return nil
}
