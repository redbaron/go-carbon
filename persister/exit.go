package persister

import "sync"

// Exit - confirmed exit. Close channel and wait "finished" channel closed
type Exit struct {
	C  chan bool
	wg *sync.WaitGroup
}

// NewExit creates new Exit instance
func NewExit(waitCount int) *Exit {
	wg := &sync.WaitGroup{}
	wg.Add(waitCount)
	return &Exit{
		C:  make(chan bool),
		wg: wg,
	}
}

// Exit closes exit channel and wait confirmation
func (e *Exit) Exit() {
	close(e.C)
	e.wg.Wait()
}

// Done confirms exit
func (e *Exit) Done() {
	e.wg.Done()
}
