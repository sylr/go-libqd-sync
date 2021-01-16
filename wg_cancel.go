package sync

import (
	"context"
	"sync"
	"sync/atomic"
)

// CancelableWaitGroup is a Waiter that can be canceled via a context.
// If the context is canceled then Waiter.Wait() will return even if Waiter.Done()
// has not been called as much as Waiter.Add().
// Note that this waiter can only be used once. If Add() is used after the context
// has been canceled or after Done() has been called as much as Add() then it will panic.
type CancelableWaitGroup struct {
	mu      sync.Mutex
	cond    *sync.Cond
	context context.Context

	// max number of tasks
	cap int
	// current number of tasks
	cur int

	// atomic integers used as boolean
	finalized int32

	// this chan is used to release the go routine when the wait group is finalized
	// when Wait() detects that wg.cur == 0
	ch chan struct{}
}

// NewCancelableWaitGroup returns a CancelableWaitGroup which has been initialized.
func NewCancelableWaitGroup(context context.Context, cap int) *CancelableWaitGroup {
	wg := &CancelableWaitGroup{
		mu:        sync.Mutex{},
		cap:       cap,
		context:   context,
		finalized: 0,
		ch:        make(chan struct{}, 1),
	}

	wg.cond = sync.NewCond(&wg.mu)

	go func() {
		select {
		case <-wg.ch:
			// Go routine is released when wg.cur comes back to zero
		case <-wg.context.Done():
			atomic.SwapInt32(&wg.finalized, 1)
			wg.cond.Broadcast()
		}
	}()

	return wg
}

// Add adds n tasks and does not return until wg.cur + n <= wg.cap.
// However, it does return if the context is canceled.
func (wg *CancelableWaitGroup) Add(n int) {
	if n > wg.cap {
		panic("libqd/sync: trying to Add more than cap")
	}

	if atomic.LoadInt32(&wg.finalized) == 1 {
		panic("libqd/sync: wait group has been finalized and can't be re-used")
	}

	wg.mu.Lock()
	defer wg.mu.Unlock()

	for (wg.cur + n) > wg.cap {
		wg.cond.Wait()

		if atomic.LoadInt32(&wg.finalized) == 1 {
			return
		}
	}

	wg.cur += n
}

// Done decreases the number of running tasks.
func (wg *CancelableWaitGroup) Done() {
	wg.mu.Lock()
	defer wg.mu.Unlock()

	if wg.cur == 0 {
		panic("libqd/sync: called Done() more than Add()")
	}

	if atomic.LoadInt32(&wg.finalized) == 1 {
		return
	}

	wg.cur--

	wg.cond.Signal()
}

// Wait waits until the number of tasks is 0 or until the context is canceled.
func (wg *CancelableWaitGroup) Wait() {
	wg.mu.Lock()

	defer func() {
		atomic.SwapInt32(&wg.finalized, 1)
		wg.ch <- struct{}{}
		wg.mu.Unlock()
	}()

	for wg.cur > 0 {
		wg.cond.Wait()

		if atomic.LoadInt32(&wg.finalized) == 1 {
			return
		}
	}
}
