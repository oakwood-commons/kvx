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
	return nil
}
