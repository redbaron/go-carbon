package cache

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/lomik/go-carbon/points"

	"github.com/Sirupsen/logrus"
)

type queue []*queueItem

func (v queue) Len() int           { return len(v) }
func (v queue) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }
func (v queue) Less(i, j int) bool { return v[i].count < v[j].count }

// Cache stores and aggregate metrics in memory
type Cache struct {
	sync.RWMutex
	settings    *Settings
	data        map[string]*points.Points
	queue       queue
	isRunning   bool            // current state of cache worker
	inputChan   *points.Channel // from receivers
	outputChan  *points.Channel // to persisters
	queryChan   chan *Query     // from carbonlink
	exitChan    chan bool       // close for stop worker
	size        int             // points count in data
	queryCnt    int             // queries count in this checkpoint period
	overflowCnt int             // drop packages if cache full
}

// New create Cache instance and run in/out goroutine
func New() *Cache {
	settings := &Settings{
		changed:        make(chan bool),
		MaxSize:        1000000,
		GraphPrefix:    "carbon.",
		InputCapacity:  51200,
		OutputCapacity: 1024,
	}
	cache := &Cache{
		settings:    settings,
		data:        make(map[string]*points.Points, 0),
		queue:       make(queue, 0),
		isRunning:   false,
		exitChan:    make(chan bool),
		queryChan:   make(chan *Query, 1024),
		size:        0,
		queryCnt:    0,
		overflowCnt: 0,
	}
	settings.cache = cache
	return cache
}

type queueItem struct {
	metric string
	count  int
}

// doCheckpoint reorder save queue, add carbon metrics to queue
func (c *Cache) doCheckpoint() {
	c.RLock()
	graphPrefix := c.settings.GraphPrefix
	c.RUnlock()

	stat := func(metric string, value float64) {
		key := fmt.Sprintf("%scache.%s", graphPrefix, metric)
		c.add(points.NowPoint(key, value))
		c.queue = append(c.queue, &queueItem{key, 1})
	}

	start := time.Now()

	inputLenBeforeCheckpoint := c.inputChan.Len()

	newQueue := make(queue, 0)

	for key, values := range c.data {
		newQueue = append(newQueue, &queueItem{key, len(values.Data)})
	}

	sort.Sort(newQueue)

	c.queue = newQueue

	inputLenAfterCheckpoint := c.inputChan.Len()

	worktime := time.Now().Sub(start)

	stat("size", float64(c.size))
	stat("metrics", float64(len(c.data)))
	stat("queries", float64(c.queryCnt))
	stat("overflow", float64(c.overflowCnt))
	stat("checkpointTime", worktime.Seconds())
	stat("inputLenBeforeCheckpoint", float64(inputLenBeforeCheckpoint))
	stat("inputLenAfterCheckpoint", float64(inputLenAfterCheckpoint))

	logrus.WithFields(logrus.Fields{
		"time":                     worktime.String(),
		"size":                     c.size,
		"metrics":                  len(c.data),
		"queries":                  c.queryCnt,
		"overflow":                 c.overflowCnt,
		"inputLenBeforeCheckpoint": inputLenBeforeCheckpoint,
		"inputLenAfterCheckpoint":  inputLenAfterCheckpoint,
		"inputCapacity":            c.inputChan.Size(),
	}).Info("[cache] doCheckpoint()")

	c.queryCnt = 0
	c.overflowCnt = 0
}

func (c *Cache) worker() {
	var values *points.Points
	var sendTo chan *points.Points

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	var maxSize int
	var settingsChanged chan bool

	refreshSettings := func() {
		c.RLock()
		defer c.RUnlock()

		settingsChanged = c.settings.changed
		maxSize = c.settings.MaxSize
	}

	refreshSettings()

	// Call Out() and In() for create channels if nil
	out, outChanged := c.Out().Current()
	in, inChanged := c.In().Current()

	for {
		if values == nil {
			values = c.pop()
		}

		if values == nil {
			sendTo = nil
		} else {
			sendTo = out
		}

		select {
		// checkpoint
		case <-ticker.C:
			c.doCheckpoint()

		// changed input channel
		case <-inChanged:
			in, inChanged = c.inputChan.Current()

		// changed output channel
		case <-outChanged:
			out, outChanged = c.outputChan.Current()

		// carbonlink
		case query := <-c.queryChan:
			c.queryCnt++
			reply := NewReply()

			if values != nil && values.Metric == query.Metric {
				reply.Points = values.Copy()
			} else if v, ok := c.data[query.Metric]; ok {
				reply.Points = v.Copy()
			}

			query.ReplyChan <- reply

		// settings updated
		case <-settingsChanged:
			refreshSettings()

		// to persister
		case sendTo <- values:
			values = nil

		// from receiver
		case msg := <-in:
			if maxSize == 0 || c.size < maxSize {
				c.add(msg)
			} else {
				c.overflowCnt++
			}

		// exit
		case <-c.exitChan:
			break
		}
	}

}

// In returns input channel
func (c *Cache) In() *points.Channel {
	c.RLock()
	defer c.RUnlock()

	if c.inputChan == nil {
		c.inputChan = points.NewChannel(c.settings.InputCapacity)
	}
	return c.inputChan
}

// Out returns output channel
func (c *Cache) Out() *points.Channel {
	c.RLock()
	defer c.RUnlock()

	if c.outputChan == nil {
		c.outputChan = points.NewChannel(c.settings.OutputCapacity)
	}
	return c.outputChan
}

// Query returns carbonlink query channel
func (c *Cache) Query() chan *Query {
	return c.queryChan
}

// Start worker
func (c *Cache) Start() {
	go c.worker()
}

// Stop worker
func (c *Cache) Stop() {
	close(c.exitChan)
}
