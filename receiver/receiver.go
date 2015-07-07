package receiver

import (
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/lomik/go-carbon/points"
)

type rcvType int

const (
	typeTCP rcvType = 1 + iota
	typeUDP
	typePICKLE
)

// Receiver is base receiver
type Receiver struct {
	settings           *Settings
	rcvType            rcvType
	addr               net.Addr
	out                *points.Channel
	exit               chan bool
	metricsReceived    uint32 // received counter
	errors             uint32 // errors counter
	active             int32  // tcp, pickle. current connected
	incompleteReceived uint32 // udp. messages chunked by MTU
}

// new create new instance of Receiver
func new(out *points.Channel) *Receiver {
	settings := &Settings{
		GraphPrefix:   "carbon.",
		LogIncomplete: false,
		changed:       make(chan bool),
	}
	rcv := &Receiver{
		settings: settings,
		out:      out,
		exit:     make(chan bool),
	}
	settings.rcv = rcv
	return rcv
}

// Addr returns binded socket address. For bind port 0 in tests
func (rcv *Receiver) Addr() net.Addr {
	return rcv.addr
}

// Stop all listeners
func (rcv *Receiver) Stop() {
	close(rcv.exit)
}

// TypeString return "udp", "tcp", "pickle"
func (rcv *Receiver) TypeString() string {
	switch rcv.rcvType {
	case typeTCP:
		return "tcp"
	case typeUDP:
		return "udp"
	case typePICKLE:
		return "pickle"
	}
	return "unknown"
}

// doCheckpoint sends internal statistics to cache
func (rcv *Receiver) doCheckpoint(statChan chan<- *points.Points, graphPrefix string) {
	protocolPrefix := rcv.TypeString()

	stat := func(metric string, value float64) {
		key := fmt.Sprintf("%s%s.%s", graphPrefix, protocolPrefix, metric)
		statChan <- points.NowPoint(key, value)
	}

	statAtomicUint32 := func(metric string, addr *uint32) {
		value := atomic.LoadUint32(addr)
		atomic.AddUint32(addr, -value)
		stat(metric, float64(value))
	}

	statAtomicUint32("metricsReceived", &rcv.metricsReceived)
	statAtomicUint32("errors", &rcv.errors)

	if rcv.rcvType == typeTCP || rcv.rcvType == typePICKLE {
		stat("active", float64(atomic.LoadInt32(&rcv.active)))
	}

	if rcv.rcvType == typeUDP {
		statAtomicUint32("incompleteReceived", &rcv.incompleteReceived)
	}
}

func (rcv *Receiver) checkpointWorker() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	var graphPrefix string
	var settingsChanged chan bool

	refreshSettings := func() {
		rcv.settings.RLock()
		defer rcv.settings.RUnlock()

		settingsChanged = rcv.settings.changed
		graphPrefix = rcv.settings.GraphPrefix
	}

	refreshSettings()

	out, outChanged := rcv.out.In()

	for {
		select {
		case <-ticker.C:
			rcv.doCheckpoint(out, graphPrefix)

		// settings updated
		case <-settingsChanged:
			refreshSettings()

		// changed output channel
		case <-outChanged:
			out, outChanged = rcv.out.In()

		case <-rcv.exit:
			return
		}
	}
}
