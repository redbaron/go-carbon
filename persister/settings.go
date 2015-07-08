package persister

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

// Validate ...
func (s *Settings) Validate() error {
	return nil
}
