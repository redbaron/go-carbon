package receiver

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"github.com/Sirupsen/logrus"
	"github.com/lomik/go-carbon/points"
)

type rcvType int

const (
	typeTCP rcvType = 1 + iota
	typeUDP
	typePICKLE
)

// Settings unified for TCP, Pickle and UDP receivers. Has all settings for all receivers
type Settings struct {
	GraphPrefix   string // prefix for internal metrics
	LogIncomplete bool   // log incomplete messages in UDP receiver
}

// Receiver is base receiver
type Receiver struct {
	sync.RWMutex
	settings           *Settings
	settingsChanged    chan bool
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
	}
	return &Receiver{
		settings:        settings,
		settingsChanged: make(chan bool),
		out:             out,
		exit:            make(chan bool),
	}
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

// EditSettings calls callback with settings instance. Not raises any error on change settings timeout
func (rcv *Receiver) EditSettings(callback func(*Settings)) {
	settings := rcv.Settings(nil)
	if settings != nil {
		callback(settings)
		rcv.Settings(settings)
	}
}

// Settings returns copy of cache settings object
func (rcv *Receiver) Settings(newSettings *Settings) *Settings {

	if newSettings == nil { // read-only
		rcv.RLock()
		defer rcv.RUnlock()

		s := *rcv.settings
		return &s
	}

	rcv.Lock()
	defer rcv.Unlock()

	// change settings here
	if newSettings.GraphPrefix != rcv.settings.GraphPrefix {
		logrus.WithFields(logrus.Fields{
			"old": rcv.settings.GraphPrefix,
			"new": newSettings.GraphPrefix,
		}).Infof("[%s] cache.GraphPrefix changed", rcv.TypeString())

		rcv.settings.GraphPrefix = newSettings.GraphPrefix
	}

	if newSettings.LogIncomplete != rcv.settings.LogIncomplete {
		logrus.WithFields(logrus.Fields{
			"old": rcv.settings.LogIncomplete,
			"new": newSettings.LogIncomplete,
		}).Infof("[%s] %s.LogIncomplete changed", rcv.TypeString(), rcv.TypeString())

		rcv.settings.LogIncomplete = newSettings.LogIncomplete
	}

	changed := rcv.settingsChanged
	rcv.settingsChanged = make(chan bool)
	close(changed)

	s := *rcv.settings

	return &s
}

// doCheckpoint sends internal statistics to cache
func (rcv *Receiver) doCheckpoint() {
	rcv.RLock()
	graphPrefix := rcv.settings.GraphPrefix
	rcv.RUnlock()

	protocolPrefix := rcv.TypeString()

	statChan := rcv.out.Chan()

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
