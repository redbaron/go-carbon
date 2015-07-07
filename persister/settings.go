package persister

import "sync"

// Settings of whisper persister
type Settings struct {
	sync.RWMutex
	changed             chan bool // subscribe to channel for notify about changed settings
	persister           *Whisper  // for apply new settings
	Enabled             bool      // can be disabled
	GraphPrefix         string    // prefix for internal metrics
	RootPath            string    // root directory for store *.wsp
	Workers             int       // save to whisper workers count
	MaxUpdatesPerSecond int       // throttling
	SchemasFile         string
	AggregationFile     string
	schemas             *WhisperSchemas
	aggregation         *WhisperAggregation
}

// Copy returns copy of settings object
func (s *Settings) Copy() *Settings {
	s.RLock()
	defer s.RUnlock()

	c := *s
	return &c
}

// Settings returns copy of cache settings object
func (p *Whisper) Settings() *Settings {
	return p.settings.Copy()
}

// Validate ...
func (s *Settings) Validate() error {
	return nil
}
