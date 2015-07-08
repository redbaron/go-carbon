package persister

import "fmt"

// Settings of whisper persister
type Settings struct {
	Enabled             bool   // can be disabled
	GraphPrefix         string // prefix for internal metrics
	RootPath            string // root directory for store *.wsp
	Workers             int    // save to whisper workers count
	MaxUpdatesPerSecond int    // throttling
	SchemasFile         string
	AggregationFile     string
	schemas             *WhisperSchemas
	aggregation         *WhisperAggregation
}

// NewSettings create new Settings instance
func NewSettings() *Settings {
	return &Settings{
		Enabled:     false,
		GraphPrefix: "carbon.",
		Workers:     1,
	}
}

// Copy returns copy of settings object
func (s *Settings) Copy() *Settings {
	c := *s
	return &c
}

// Settings returns copy of cache settings object
func (p *Whisper) Settings() *Settings {
	return p.settings.Copy()
}

// LoadAndValidate loads schemas and aggregation from files. Validate settings
func (s *Settings) LoadAndValidate() error {
	var err error

	if !s.Enabled { // if disabled no validate other settings
		return nil
	}

	if s.SchemasFile == "" {
		return fmt.Errorf("schemas file is empty")
	}

	s.schemas, err = ReadWhisperSchemas(s.SchemasFile)
	if err != nil {
		return err
	}

	if s.AggregationFile != "" {
		s.aggregation, err = ReadWhisperAggregation(s.AggregationFile)
		if err != nil {
			return err
		}
	} else {
		s.aggregation = NewWhisperAggregation()
	}

	return nil
}

// IsChanged returns true if settings differ
func (s *Settings) IsChanged(other *Settings) bool {
	return false
}
