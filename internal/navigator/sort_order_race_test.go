package navigator

import (
	"sync"
	"testing"
)

// TestSortOrderConcurrency_Race hammers SetSortOrder/NodeToRows (the legacy
// global-state API) alongside NodeToRowsWith (the explicit-order API) from
// many goroutines. Regression test for issue #83.
func TestSortOrderConcurrency_Race(t *testing.T) {
	node := map[string]interface{}{"b": 1, "a": 2, "c": 3}

	var wg sync.WaitGroup
	orders := []SortOrder{SortAscending, SortDescending, SortNone}

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				_ = SetSortOrder(orders[j%len(orders)])
				_ = NodeToRows(node)
			}
		}()
	}

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				_ = NodeToRowsWith(node, orders[j%len(orders)])
			}
		}()
	}

	wg.Wait()
}

// TestNodeToRowsWith_ConcurrentDifferentOrders confirms NodeToRowsWith
// returns the requested order even when called concurrently with a
// different order in another goroutine (no shared global state).
func TestNodeToRowsWith_ConcurrentDifferentOrders(t *testing.T) {
	node := map[string]interface{}{"b": 1, "a": 2, "c": 3}

	var wg sync.WaitGroup
	errs := make(chan string, 200)

	check := func(order SortOrder, wantFirstKey string) {
		defer wg.Done()
		rows := NodeToRowsWith(node, order)
		if len(rows) == 0 || rows[0][0] != wantFirstKey {
			errs <- "unexpected row order for " + string(order)
		}
	}

	for i := 0; i < 100; i++ {
		wg.Add(2)
		go check(SortAscending, "a")
		go check(SortDescending, "c")
	}
	wg.Wait()
	close(errs)

	for e := range errs {
		t.Error(e)
	}
}
