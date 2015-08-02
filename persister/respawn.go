package persister

import "github.com/lomik/go-carbon/points"

// Respawn stops old whisper and start new if settings is changed
func Respawn(p *Whisper, settings *Settings, inChan *points.Channel) *Whisper {
	if p == nil || p.Settings().IsChanged(settings) {
		if p != nil {
			p.Stop()
		}
		p = NewWhisper(inChan, settings)
		p.Start()
	}
	return p
}
