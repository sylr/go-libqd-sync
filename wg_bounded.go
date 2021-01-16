package sync

import (
	"sync"
)

// NewWaiter returns a Waiter, a sync.WaitGroup if cap i <= 0, a BoundedWaitGroup otherwise.
func NewWaiter(cap int) Waiter {
	if cap > 0 {
		wg := NewBoundedWaitGroup(cap)
		return &wg
	}

	return &sync.WaitGroup{}
}

// BoundedWaitGroup is a wait group which has a limit boundary meaning it will
// wait for Done() to be called before releasing Add(n) if the limit has been reached.
type BoundedWaitGroup struct {
	wg sync.WaitGroup
	ch chan struct{}
}

// NewBoundedWaitGroup returns a new BoundedWaitGroup
func NewBoundedWaitGroup(cap int) BoundedWaitGroup {
	return BoundedWaitGroup{ch: make(chan struct{}, cap)}
}

// Add adds delta, which may be negative, to the BoundedWaitGroup counter.
// If counter + delta is greater that the cap of the BoundedWaitGroup then Add
// would block until there is enough slots made available in the the WaitGroup.
func (bwg *BoundedWaitGroup) Add(delta int) {
	if delta > 0 {
		for i := 0; i < delta; i++ {
			bwg.ch <- struct{}{}
		}
	} else {
		for i := 0; i > delta; i-- {
			<-bwg.ch
		}
	}

	bwg.wg.Add(delta)
}

// Done decrements the WaitGroup counter by one.
func (bwg *BoundedWaitGroup) Done() {
	bwg.Add(-1)
}

// Wait blocks until the WaitGroup counter is zero.
func (bwg *BoundedWaitGroup) Wait() {
	bwg.wg.Wait()
}
