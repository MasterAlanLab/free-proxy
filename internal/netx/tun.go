package netx

import (
	"errors"
	"fmt"
	"sync"
)

// TunAllocator hands out exclusive tunN device names from a pool for concurrent
// probing, mirroring the former asyncio allocator.
type TunAllocator struct {
	start, end int
	mu         sync.Mutex
	allocated  map[int]bool
}

// NewTunAllocator creates an allocator over [start, end].
func NewTunAllocator(start, end int) (*TunAllocator, error) {
	if start > end {
		return nil, errors.New("TUN allocation start must not exceed end")
	}
	return &TunAllocator{start: start, end: end, allocated: map[int]bool{}}, nil
}

// Allocate reserves a device and returns its name plus a release func. The
// release func is safe to call once; typically deferred.
func (a *TunAllocator) Allocate() (device string, release func(), err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i := a.start; i <= a.end; i++ {
		if !a.allocated[i] {
			a.allocated[i] = true
			idx := i
			return fmt.Sprintf("tun%d", idx), func() {
				a.mu.Lock()
				delete(a.allocated, idx)
				a.mu.Unlock()
			}, nil
		}
	}
	return "", nil, errors.New("no test TUN devices are available")
}
