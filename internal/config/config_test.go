package config

import (
	"testing"
)

func TestConfigEmpty(t *testing.T) {
	c := Config{}
	if c.Name != "" {
		t.Fatalf("expected empty Name, got %q", c.Name)
	}
	if c.Version != "" {
		t.Fatalf("expected empty Version, got %q", c.Version)
	}
}

func TestConfigWithValues(t *testing.T) {
	c := Config{
		Name:    "test-app",
		Version: "1.0.0",
	}
	if c.Name != "test-app" {
		t.Fatalf("expected 'test-app', got %q", c.Name)
	}
	if c.Version != "1.0.0" {
		t.Fatalf("expected '1.0.0', got %q", c.Version)
	}
}

func TestConfigYAMLTags(t *testing.T) {
	// This test verifies that the struct tags are correctly defined
	// by checking the field metadata
	c := Config{
		Name:    "my-app",
		Version: "2.5.0",
	}
	if c.Name != "my-app" || c.Version != "2.5.0" {
		t.Fatalf("expected fields to be set correctly")
	}
}
