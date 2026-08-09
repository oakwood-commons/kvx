package ui

import (
	"sync"
	"testing"
)

// TestThemeConcurrency_Race hammers the theme globals (currentTheme,
// loadedThemes, ThemePresets) from many goroutines simultaneously. It is a
// regression test for issue #83: before themeMu was introduced, this test
// reliably tripped `-race` (concurrent map read/write on loadedThemes, plus
// a write-on-read-path race on currentTheme via CurrentTheme()).
func TestThemeConcurrency_Race(t *testing.T) {
	cfg := &ThemeConfigFile{
		Themes: map[string]ThemeConfig{
			"dark":  {},
			"light": {BorderStyle: "rounded"},
		},
	}
	if err := InitializeThemes(cfg); err != nil {
		t.Fatalf("InitializeThemes: %v", err)
	}

	var readerWG, writerWG sync.WaitGroup
	stop := make(chan struct{})

	// Readers.
	for i := 0; i < 8; i++ {
		readerWG.Add(1)
		go func() {
			defer readerWG.Done()
			for {
				select {
				case <-stop:
					return
				default:
					_ = CurrentTheme()
					_, _ = GetTheme("dark")
					_ = GetAvailableThemes()
				}
			}
		}()
	}

	// Writers.
	names := []string{"dark", "light"}
	for i := 0; i < 4; i++ {
		writerWG.Add(1)
		go func() {
			defer writerWG.Done()
			for j := 0; j < 200; j++ {
				_ = SetThemeByName(names[j%len(names)])
			}
		}()
	}

	// Concurrent re-initialization (exercises the loadedThemes/ThemePresets
	// map-replacement path, including the DefaultTheme-before-lock ordering
	// that avoids self-deadlock).
	for i := 0; i < 2; i++ {
		writerWG.Add(1)
		go func() {
			defer writerWG.Done()
			for j := 0; j < 50; j++ {
				_ = InitializeThemes(cfg)
			}
		}()
	}

	writerWG.Wait()
	close(stop)
	readerWG.Wait()
}
