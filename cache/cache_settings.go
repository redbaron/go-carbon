package cache

import (
	"sync"

	"github.com/Sirupsen/logrus"
)

// Settings ...
type Settings struct {
	sync.RWMutex
	settingsChanged chan bool // subscribe to channel for notify about changed settings
	cache           *Cache    // for apply new settings
	MaxSize         int       // cache capacity (points)
	GraphPrefix     string    // prefix for internal metrics
	InputCapacity   int       // input channel capacity
	OutputCapacity  int       // output channel capacity
}

// Copy returns copy of settings object
func (s *Settings) Copy() *Settings {
	s.RLock()
	defer s.RUnlock()

	c := *s
	return &c
}

// Validate ...
func (s *Settings) Validate() error {
	return nil
}

// Apply ...
func (s *Settings) Apply() error {
	if err := s.Validate(); err != nil {
		return err
	}

	cache := s.cache
	obj := cache.settings

	obj.Lock()
	defer obj.Unlock()

	if s.MaxSize != obj.MaxSize {
		logrus.Infof("[cache] cache.MaxSize changed: %#v -> %#v", obj.MaxSize, s.MaxSize)
		obj.MaxSize = s.MaxSize
	}

	if s.GraphPrefix != obj.GraphPrefix {
		logrus.Infof("[cache] cache.GraphPrefix changed: %#v -> %#v", obj.GraphPrefix, s.GraphPrefix)
		obj.GraphPrefix = s.GraphPrefix
	}

	if s.InputCapacity != obj.InputCapacity {
		logrus.Infof("[cache] cache.InputCapacity changed: %#v -> %#v", obj.InputCapacity, s.InputCapacity)

		obj.InputCapacity = s.InputCapacity
		if cache.inputChan != nil {
			cache.inputChan.Resize(obj.InputCapacity)
		}
	}

	if s.OutputCapacity != obj.OutputCapacity {
		logrus.Infof("[cache] cache.OutputCapacity changed: %#v -> %#v", obj.OutputCapacity, s.OutputCapacity)

		obj.OutputCapacity = s.OutputCapacity

		if cache.outputChan != nil {
			cache.outputChan.Resize(obj.OutputCapacity)
		}
	}

	return nil
}

// Settings returns copy of cache settings object
func (c *Cache) Settings() *Settings {
	return c.settings.Copy()
}
