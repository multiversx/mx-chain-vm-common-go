package builtInFunctions

import "sync"

// DrwaCounterSet is a thread-safe uint64 counter map shared by DRWA gate
// metrics and sync-layer observability. It caps at MaxUint64 to prevent
// wrap-to-zero that would mislead monitoring.
type DrwaCounterSet struct {
	mut      sync.Mutex
	counters map[string]uint64
}

func NewDrwaCounterSet() *DrwaCounterSet {
	return &DrwaCounterSet{counters: make(map[string]uint64)}
}

func (c *DrwaCounterSet) Increment(metric string) {
	c.mut.Lock()
	if c.counters[metric] < ^uint64(0) {
		c.counters[metric]++
	}
	c.mut.Unlock()
}

func (c *DrwaCounterSet) Snapshot() map[string]uint64 {
	c.mut.Lock()
	defer c.mut.Unlock()
	out := make(map[string]uint64, len(c.counters))
	for k, v := range c.counters {
		out[k] = v
	}
	return out
}

func (c *DrwaCounterSet) Reset() {
	c.mut.Lock()
	c.counters = make(map[string]uint64)
	c.mut.Unlock()
}
