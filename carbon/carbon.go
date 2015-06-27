package carbon

import (
	"github.com/lomik/go-carbon/cache"
	"github.com/lomik/go-carbon/persister"
	"github.com/lomik/go-carbon/receiver"
)

// Carbon - main application controller
type Carbon struct {
	Cache     *cache.Cache
	UDP       *receiver.Receiver
	TCP       *receiver.Receiver
	Pickle    *receiver.Receiver
	Persister *persister.Whisper
}

// New returns new instance of Carbon
func New() *Carbon {
	return &Carbon{}
}

// Configure init or change carbon configuration
func (carbon *Carbon) Configure(config *Config) {

}
