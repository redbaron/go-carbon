package points

import (
	"sync"
	"time"
)

// Channel is resizable *Point channel
type Channel struct {
	sync.RWMutex
	in              chan *Points
	inChanged       chan bool
	out             chan *Points
	outChanged      chan bool
	closeOldTimeout time.Duration
	exit            chan bool
	exitThrottling  chan bool // close for stop throttling worker
	size            int       // channel size
	ratePerSec      int       // throttling
}

// NewChannel creates new channel
func NewChannel(size int) *Channel {
	ch := make(chan *Points, size)

	return &Channel{
		in:              ch,
		out:             ch,
		inChanged:       make(chan bool),
		outChanged:      make(chan bool),
		closeOldTimeout: time.Duration(5 * time.Minute),
		exit:            make(chan bool),
		exitThrottling:  make(chan bool),
		size:            size,
		ratePerSec:      0,
	}
}

// In returns pair of points channel and changed channel
func (c *Channel) In() (chan *Points, chan bool) {
	c.RLock()
	defer c.RUnlock()

	return c.in, c.inChanged
}

// Out returns pair of points channel and changed channel
func (c *Channel) Out() (chan *Points, chan bool) {
	c.RLock()
	defer c.RUnlock()

	return c.out, c.outChanged
}

// read all messages from old channel and close it after timeout
func (c *Channel) quarantine(in chan *Points) {

	var p *Points
	var opened bool
	var sendTo chan *Points
	var recvFrom chan *Points

	out, changeOut := c.In()

	// check timeout every minute
	ticker := time.NewTicker(c.closeOldTimeout)
	defer ticker.Stop()

	var activityCounter int
	var prevActivityCounter int

	for {
		if p == nil {
			sendTo = nil
			recvFrom = in
		} else {
			sendTo = out
			recvFrom = nil
		}

		select {
		// check timeout
		case <-ticker.C:
			if p == nil && activityCounter == prevActivityCounter {
				close(in)
				return
			}
			prevActivityCounter = activityCounter
		// changed out channel
		case <-changeOut:
			out, changeOut = c.In()
		// send message to output
		case sendTo <- p:
			p = nil
			activityCounter++
		// receive new message from input
		case p, opened = <-recvFrom:
			if !opened {
				return
			}
			activityCounter++
		}

	}
}

// Size returns current size of channel
func (c *Channel) Size() int {
	ch, _ := c.In()
	return cap(ch)
}

// Len return current "len" of active channel
func (c *Channel) Len() int {
	ch, _ := c.In()
	return len(ch)
}

// InChan return current IN channel. With mutex lock
func (c *Channel) InChan() chan *Points {
	ch, _ := c.In()
	return ch
}

// OutChan return current OUT channel. With mutex lock
func (c *Channel) OutChan() chan *Points {
	ch, _ := c.Out()
	return ch
}

func (c *Channel) changeChannel(newIn chan *Points, newOut chan *Points) {
	oldIn := c.in
	oldInChanged := c.inChanged
	oldOut := c.out
	oldOutChanged := c.outChanged

	c.in = newIn
	c.inChanged = make(chan bool)
	c.out = newOut
	c.outChanged = make(chan bool)

	close(oldInChanged)
	close(oldOutChanged)

	go c.quarantine(oldIn)
	if oldIn != oldOut {
		go c.quarantine(oldOut)
	}
}

// apply settings
func (c *Channel) apply() {
	newIn := make(chan *Points, c.size)
	newOut := newIn
	if c.ratePerSec != 0 {
		newOut = make(chan *Points, c.size)
	}

	exitThrottling := c.exitThrottling
	c.exitThrottling = make(chan bool)
	close(exitThrottling)

	c.changeChannel(newIn, newOut)

	if c.ratePerSec != 0 {
		go throttleWorker(c.ratePerSec, newIn, newOut, c.exitThrottling)
	}
}

func throttleWorker(rps int, in chan *Points, out chan *Points, exit chan bool) {
	step := time.Duration(1e9/rps) * time.Nanosecond

	var p *Points
	var opened bool

	// start flight
	throttleTicker := time.NewTicker(step)
	defer throttleTicker.Stop()

	for {
		select {
		// exit
		case <-exit:
			return
		// receive
		case <-throttleTicker.C:
			if p, opened = <-in; !opened {
				return
			}
			out <- p
		}
	}
}

// Resize channel
func (c *Channel) Resize(newSize int) {
	c.Lock()
	defer c.Unlock()

	c.size = newSize

	c.apply()
}

// Throttle enable or disable throttling
func (c *Channel) Throttle(ratePerSec int) {
	c.Lock()
	defer c.Unlock()

	c.ratePerSec = ratePerSec
	c.apply()
}
