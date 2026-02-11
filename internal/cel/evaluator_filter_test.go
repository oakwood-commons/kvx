package cel

import "testing"

func TestDiscoverCELFunctionDocsFiltersInternalMacros(t *testing.T) {
	// Set example hints to exercise hint integration
	SetExampleHints(map[string]string{"filter": "e.g. items.filter(x, x.ok)"})
	defer SetExampleHints(nil)

	funcs, err := DiscoverCELFunctionDocs()
	if err != nil {
		t.Fatalf("DiscoverCELFunctionDocs error: %v", err)
	}
	for _, f := range funcs {
		if len(f) > 0 && f[0] == '@' {
			t.Fatalf("internal macro leaked into suggestions: %q", f)
		}
	}
}
