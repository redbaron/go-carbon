package carbon

import (
	"os"
	"strings"
	"sync"

	"github.com/lomik/go-carbon/cache"
	"github.com/lomik/go-carbon/persister"
	"github.com/lomik/go-carbon/receiver"
)

// Carbon - main application controller
type Carbon struct {
	sync.RWMutex
	Cache      *cache.Cache
	Carbonlink *cache.CarbonlinkListener
	UDP        *receiver.Receiver
	TCP        *receiver.Receiver
	Pickle     *receiver.Receiver
	Persister  *persister.Whisper
}

// New returns new instance of Carbon
func New() *Carbon {
	core := cache.New()
	core.Start()

	return &Carbon{
		Cache:      core,
		Carbonlink: cache.NewCarbonlinkListener(core.Query()),
		UDP:        receiver.NewUDP(core.In()),
		TCP:        receiver.NewTCP(core.In()),
		Pickle:     receiver.NewPickle(core.In()),
	}
}

// Configure init or change carbon configuration
func (app *Carbon) Configure(config *Config, apply bool) error {
	app.Lock()
	defer app.Unlock()

	// carbon-cache prefix
	if hostname, err := os.Hostname(); err == nil {
		hostname = strings.Replace(hostname, ".", "_", -1)
		config.Common.GraphPrefix = strings.Replace(config.Common.GraphPrefix, "{host}", hostname, -1)
	} else {
		config.Common.GraphPrefix = strings.Replace(config.Common.GraphPrefix, "{host}", "localhost", -1)
	}

	cacheSettings := app.Cache.Settings()
	udpSettings := app.UDP.Settings()
	tcpSettings := app.TCP.Settings()
	pickleSettings := app.Pickle.Settings()
	persisterSettings := persister.NewSettings()

	// core settings
	cacheSettings.GraphPrefix = config.Common.GraphPrefix
	cacheSettings.InputCapacity = config.Cache.InputBuffer
	cacheSettings.MaxSize = config.Cache.MaxSize

	// listeners settings
	udpSettings.GraphPrefix = config.Common.GraphPrefix
	udpSettings.Enabled = config.UDP.Enabled
	udpSettings.LogIncomplete = config.UDP.LogIncomplete
	udpSettings.ListenAddr = config.UDP.Listen

	tcpSettings.GraphPrefix = config.Common.GraphPrefix
	tcpSettings.Enabled = config.TCP.Enabled
	tcpSettings.ListenAddr = config.TCP.Listen

	pickleSettings.GraphPrefix = config.Common.GraphPrefix
	pickleSettings.Enabled = config.Pickle.Enabled
	pickleSettings.ListenAddr = config.Pickle.Listen

	// persister
	persisterSettings.Enabled = config.Whisper.Enabled
	persisterSettings.GraphPrefix = config.Common.GraphPrefix
	persisterSettings.RootPath = config.Whisper.DataDir
	persisterSettings.Workers = config.Whisper.Workers
	persisterSettings.MaxUpdatesPerSecond = config.Whisper.MaxUpdatesPerSecond
	persisterSettings.SchemasFile = config.Whisper.Schemas
	persisterSettings.AggregationFile = config.Whisper.Aggregation

	var err error
	var tmpErr error

	// validate all. Fail on first error
	if err = cacheSettings.Validate(); err != nil {
		return err
	}
	if err = udpSettings.Validate(); err != nil {
		return err
	}
	if err = tcpSettings.Validate(); err != nil {
		return err
	}
	if err = pickleSettings.Validate(); err != nil {
		return err
	}
	if err = persisterSettings.LoadAndValidate(); err != nil {
		return err
	}

	if !apply {
		return nil
	}

	// apply all. Fail after all applied (if can)
	if tmpErr = cacheSettings.Apply(); tmpErr != nil {
		err = tmpErr
	}
	if tmpErr = udpSettings.Apply(); tmpErr != nil {
		err = tmpErr
	}
	if tmpErr = tcpSettings.Apply(); tmpErr != nil {
		err = tmpErr
	}
	if tmpErr = pickleSettings.Apply(); tmpErr != nil {
		err = tmpErr
	}

	// if persister settings changed RESTART THEM
	if app.Persister == nil || app.Persister.Settings().IsChanged(persisterSettings) {
		if app.Persister != nil {
			app.Persister.Stop()
		}
		app.Persister = persister.NewWhisper(app.Cache.Out(), persisterSettings)
		app.Persister.Start()
	}

	return err
}
