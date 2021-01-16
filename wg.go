package sync

// Waiter interface
type Waiter interface {
	Add(int)
	Done()
	Wait()
}
