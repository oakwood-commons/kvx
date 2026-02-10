//nolint:forcetypeassert
package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"gopkg.in/yaml.v3"
)

// loadYAMLFixture loads a YAML fixture file into a generic interface{}
func loadYAMLFixture(t *testing.T) interface{} {
	t.Helper()
	p := filepath.Join("..", "..", "tests", "sample.yaml")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read fixture %s: %v", p, err)
	}
	var root interface{}
	if err := yaml.Unmarshal(data, &root); err != nil {
		t.Fatalf("yaml unmarshal: %v", err)
	}
	return root
}

// loadJSONFixture loads a JSON fixture file into a generic interface{}
func loadJSONFixture(t *testing.T, rel string) interface{} {
	t.Helper()
	p := filepath.Join("..", "..", "tests", rel)
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read fixture %s: %v", p, err)
	}
	var root interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	return root
}

// helper to create focused model with a given root
func focusedModelWithRoot(root interface{}) *Model {
	m := InitialModel(root)
	m.Root = root
	m.InputFocused = true
	m.PathInput = textinput.New()
	m.PathInput.Prompt = ""
	m.PathInput.Focus()
	return &m
}

func TestFixture_RegionsPrefixEnterNavigates(t *testing.T) {
	root := loadYAMLFixture(t)
	m := focusedModelWithRoot(root)
	// In expr mode, type full path and press Enter
	m.PathInput.SetValue("regions.asia")
	m.filterSuggestions()
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m2 := newModel.(*Model)
	if m2.Path != "_.regions.asia" {
		t.Fatalf("expected _.regions.asia, got %s", m2.Path)
	}
}

func TestFixture_CaasPrefixEnterNavigates(t *testing.T) {
	root := loadJSONFixture(t, "caas.json")
	m := focusedModelWithRoot(root)
	// In expr mode, type full key and press Enter
	m.PathInput.SetValue("cluster_001")
	m.filterSuggestions()
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m2 := newModel.(*Model)
	if m2.Path != "_.cluster_001" {
		t.Fatalf("expected _.cluster_001, got %s", m2.Path)
	}
}

// Validates real-time filtering as the user types root keys
func TestFixture_RealTimeFilteringRootKeys(t *testing.T) {
	// Switch to table mode to verify real-time filtering using type-ahead
	root := loadJSONFixture(t, "caas.json")
	m := InitialModel(root)
	m.Root = root
	m.InputFocused = false
	m.Tbl.Focus()
	// Type 'c' via type-ahead
	// v2: Use Text field for character input instead of KeyRunes + Runes
	m.Update(tea.KeyPressMsg{Text: "c", Code: 'c'})
	rows := m.Tbl.Rows()
	if len(rows) == 0 {
		t.Fatalf("expected some filtered rows for prefix 'c'")
	}
	// Type 'cl' to narrow further
	// v2: Use Text field for character input
	m.Update(tea.KeyPressMsg{Text: "l", Code: 'l'})
	rows = m.Tbl.Rows()
	if len(rows) == 0 {
		t.Fatalf("expected some filtered rows for prefix 'cl'")
	}
}

func TestFixture_CaasExactKeyEnterNavigates(t *testing.T) {
	root := loadJSONFixture(t, "caas.json")
	m := focusedModelWithRoot(root)
	// Type exact key
	m.PathInput.SetValue("cluster_001")
	m.filterSuggestions()
	// Enter should navigate exactly to pd1010
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m2 := newModel.(*Model)
	if m2.Path != "_.cluster_001" {
		t.Fatalf("expected _.cluster_001, got %s", m2.Path)
	}
}
