package cel

import (
	"sync"
	"testing"
)

// TestExampleHintsConcurrency_Race hammers SetExampleHints/GetExampleHints
// from many goroutines. Regression test for issue #83.
func TestExampleHintsConcurrency_Race(t *testing.T) {
	var wg sync.WaitGroup

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				SetExampleHints(map[string]string{"fn": "example"})
			}
		}()
	}

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				hints := GetExampleHints()
				// Mutating the returned map must not affect the package's
				// internal state (GetExampleHints returns a defensive copy).
				if hints != nil {
					hints["mutated"] = "value"
				}
			}
		}()
	}

	wg.Wait()

	hints := GetExampleHints()
	if _, ok := hints["mutated"]; ok {
		t.Fatalf("GetExampleHints leaked a mutation back into package state: %v", hints)
	}
}
