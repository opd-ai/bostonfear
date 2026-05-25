package serverengine

import (
	"sync"
	"testing"
)

func TestActionTypeTracking(t *testing.T) {
	gs, pid := newTestServer(t)

	// Track some actions
	gs.trackActionType(ActionMove)
	gs.trackActionType(ActionInvestigate)
	gs.trackActionType(ActionMove)
	gs.trackActionType(ActionCastWard)

	counters := gs.getActionTypeCounters()

	if counters[ActionMove] != 2 {
		t.Errorf("expected move count 2, got %d", counters[ActionMove])
	}
	if counters[ActionInvestigate] != 1 {
		t.Errorf("expected investigate count 1, got %d", counters[ActionInvestigate])
	}
	if counters[ActionCastWard] != 1 {
		t.Errorf("expected cast_ward count 1, got %d", counters[ActionCastWard])
	}

	_ = pid // unused but required by newTestServer
}

func TestDoomHistogramTracking(t *testing.T) {
	gs, pid := newTestServer(t)

	// Track doom levels at game end
	gs.trackDoomLevel(5)
	gs.trackDoomLevel(12)
	gs.trackDoomLevel(5)
	gs.trackDoomLevel(8)

	histogram := gs.getDoomHistogram()

	if histogram[5] != 2 {
		t.Errorf("expected doom 5 count 2, got %d", histogram[5])
	}
	if histogram[12] != 1 {
		t.Errorf("expected doom 12 count 1, got %d", histogram[12])
	}
	if histogram[8] != 1 {
		t.Errorf("expected doom 8 count 1, got %d", histogram[8])
	}

	_ = pid // unused but required by newTestServer
}

func TestActionTypeTracking_Concurrent(t *testing.T) {
	gs, _ := newTestServer(t)

	const goroutines = 32
	const incrementsPerGoroutine = 64

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			for range incrementsPerGoroutine {
				gs.trackActionType(ActionMove)
			}
		}()
	}

	wg.Wait()

	counters := gs.getActionTypeCounters()
	want := int64(goroutines * incrementsPerGoroutine)
	if counters[ActionMove] != want {
		t.Fatalf("expected move count %d, got %d", want, counters[ActionMove])
	}
}
