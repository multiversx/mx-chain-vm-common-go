package builtInFunctions

import (
	"math"
	"sync"
	"testing"
)

func TestNewDrwaCounterSet_StartsAtZero(t *testing.T) {
	cs := NewDrwaCounterSet()
	snap := cs.Snapshot()
	if len(snap) != 0 {
		t.Fatalf("expected empty snapshot from fresh counter set, got %d entries", len(snap))
	}
}

func TestDrwaCounterSet_IncrementAndSnapshot(t *testing.T) {
	cs := NewDrwaCounterSet()
	cs.Increment("alpha")
	cs.Increment("alpha")
	cs.Increment("beta")

	snap := cs.Snapshot()
	if snap["alpha"] != 2 {
		t.Fatalf("expected alpha=2, got %d", snap["alpha"])
	}
	if snap["beta"] != 1 {
		t.Fatalf("expected beta=1, got %d", snap["beta"])
	}
}

func TestDrwaCounterSet_SnapshotIsIsolatedCopy(t *testing.T) {
	cs := NewDrwaCounterSet()
	cs.Increment("x")
	snap := cs.Snapshot()

	// Mutate the snapshot; original must be unaffected.
	snap["x"] = 999
	snap["y"] = 1

	snap2 := cs.Snapshot()
	if snap2["x"] != 1 {
		t.Fatalf("snapshot mutation leaked back into counter set: x=%d", snap2["x"])
	}
	if _, exists := snap2["y"]; exists {
		t.Fatal("snapshot mutation injected phantom key into counter set")
	}
}

func TestDrwaCounterSet_ConcurrentIncrementNoLostCounts(t *testing.T) {
	cs := NewDrwaCounterSet()
	const goroutines = 50
	const incrementsPerGoroutine = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				cs.Increment("concurrent")
			}
		}()
	}
	wg.Wait()

	snap := cs.Snapshot()
	expected := uint64(goroutines * incrementsPerGoroutine)
	if snap["concurrent"] != expected {
		t.Fatalf("expected concurrent=%d, got %d (lost counts under contention)", expected, snap["concurrent"])
	}
}

func TestDrwaCounterSet_OverflowProtection(t *testing.T) {
	cs := NewDrwaCounterSet()

	// Force counter to MaxUint64 via internal access (white-box).
	cs.mut.Lock()
	cs.counters["saturated"] = math.MaxUint64
	cs.mut.Unlock()

	// Increment must cap, not wrap to zero.
	cs.Increment("saturated")

	snap := cs.Snapshot()
	if snap["saturated"] != math.MaxUint64 {
		t.Fatalf("expected saturated counter to stay at MaxUint64, got %d", snap["saturated"])
	}
}

func TestDrwaCounterSet_OverflowProtection_OneBelowMax(t *testing.T) {
	cs := NewDrwaCounterSet()

	cs.mut.Lock()
	cs.counters["almost"] = math.MaxUint64 - 1
	cs.mut.Unlock()

	cs.Increment("almost")

	snap := cs.Snapshot()
	if snap["almost"] != math.MaxUint64 {
		t.Fatalf("expected counter to reach MaxUint64, got %d", snap["almost"])
	}

	// One more increment must not wrap.
	cs.Increment("almost")
	snap = cs.Snapshot()
	if snap["almost"] != math.MaxUint64 {
		t.Fatalf("expected counter to stay at MaxUint64 after second increment, got %d", snap["almost"])
	}
}

func TestDrwaCounterSet_Reset(t *testing.T) {
	cs := NewDrwaCounterSet()
	cs.Increment("a")
	cs.Increment("b")
	cs.Reset()

	snap := cs.Snapshot()
	if len(snap) != 0 {
		t.Fatalf("expected empty snapshot after Reset, got %d entries", len(snap))
	}
}
