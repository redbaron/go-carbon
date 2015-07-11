package receiver

import (
	"sync"

	"github.com/Sirupsen/logrus"
)

// Settings unified for TCP, Pickle and UDP receivers. Has all settings for all receivers
type Settings struct {
	sync.RWMutex
	changed       chan bool // subscribe to channel for notify about changed settings
	rcv           *Receiver // for apply new settings
	Enabled       bool      // any type of listener can be disabled
	GraphPrefix   string    // prefix for internal metrics
	LogIncomplete bool      // log incomplete messages in UDP receiver
	ListenAddr    string    // network address for listener
}

// Copy returns copy of settings object
func (s *Settings) Copy() *Settings {
	s.RLock()
	defer s.RUnlock()

	c := *s
	return &c
}

// Settings returns copy of cache settings object
func (rcv *Receiver) Settings() *Settings {
	return rcv.settings.Copy()
}

// Validate ...
func (s *Settings) Validate() error {
	return nil
}

// Apply ...
func (s *Settings) Apply() error {
	var err error
	if err = s.Validate(); err != nil {
		return err
	}

	rcv := s.rcv
	obj := rcv.settings

	obj.Lock()
	defer obj.Unlock()

	if s.GraphPrefix != obj.GraphPrefix {
		logrus.Infof("[%s] GraphPrefix changed: %#v -> %#v", rcv.TypeString(), obj.GraphPrefix, s.GraphPrefix)
		obj.GraphPrefix = s.GraphPrefix
	}

	if s.LogIncomplete != obj.LogIncomplete {
		logrus.Infof("[%s] LogIncomplete changed: %#v -> %#v", rcv.TypeString(), obj.LogIncomplete, s.LogIncomplete)
		obj.LogIncomplete = s.LogIncomplete
	}

	if rcv.rcvType == typeUDP && s.LogIncomplete != obj.LogIncomplete {
		logrus.Infof("[%s] LogIncomplete changed: %#v -> %#v", rcv.TypeString(), obj.LogIncomplete, s.LogIncomplete)
		obj.LogIncomplete = s.LogIncomplete
	}

	if s.Enabled != obj.Enabled {
		logrus.Infof("[%s] Enabled changed: %#v -> %#v", rcv.TypeString(), obj.Enabled, s.Enabled)
		obj.Enabled = s.Enabled
	}

	// @TODO: ListenAddr changed

	if obj.Enabled && rcv.Addr() == nil { // start if stopped
		logrus.Debugf("[%s] Start listener", rcv.TypeString())
		err = rcv.start()
	}

	if !obj.Enabled && rcv.Addr() != nil { // stop if running
		logrus.Debugf("[%s] Stop listener", rcv.TypeString())
		rcv.Stop()
	}

	changed := obj.changed
	obj.changed = make(chan bool)
	close(changed)

	if err != nil {
		logrus.Debugf("[%s] settings.Apply() error: %s", rcv.TypeString(), err.Error())
	}

	return err
}
