package points

import (
	"sync"
	"time"
)

// Channel is resizable *Point channel
type Channel struct {
	sync.RWMutex
	active          chan *Points
	changed         chan bool
	closeOldTimeout time.Duration
}

// NewChannel creates new channel
func NewChannel(size int) *Channel {
	return &Channel{
		active:          make(chan *Points, size),
		changed:         make(chan bool),
		closeOldTimeout: time.Duration(5 * time.Minute),
	}
}

// Current returns pair of points channel and changed channel
func (c *Channel) Current() (chan *Points, chan bool) {
	c.RLock()
	defer c.RUnlock()

	return c.active, c.changed
}

// read all messages from old channel and close it after timeout
func (c *Channel) quarantine(in chan *Points) {

	var p *Points
	var opened bool
	var sendTo chan *Points
	var recvFrom chan *Points

	out, changeOut := c.Current()

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
			out, changeOut = c.Current()
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

func (c *Channel) changeChannel(newChannel chan *Points) {
	c.Lock()

	oldChannel := c.active
	oldChanged := c.changed

	c.active = newChannel
	c.changed = make(chan bool)

	c.Unlock()

	close(oldChanged)
	go c.quarantine(oldChannel)
}

// Resize channel
func (c *Channel) Resize(newSize int) {
	newChannel := make(chan *Points, newSize)
	c.changeChannel(newChannel)
}

// Size returns current size of channel
func (c *Channel) Size() int {
	ch, _ := c.Current()
	return len(ch)
}
