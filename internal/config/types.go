package config

// Config represents optional top-level metadata for displaying in the UI header
// Only Name and Version are used; all other fields are ignored and navigation
// works with the generic interface{} representation
type Config struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}
