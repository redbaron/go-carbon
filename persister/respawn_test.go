package persister

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"testing"
	"time"

	"github.com/lomik/go-carbon/helper"
	"github.com/lomik/go-carbon/points"
	"github.com/stretchr/testify/assert"
)

func newTestSettings(t *testing.T, dataDir string) *Settings {
	assert := assert.New(t)

	assert.NoError(os.Mkdir(dataDir, 0755))

	settings := NewSettings()
	settings.RootPath = dataDir
	settings.SchemasFile = path.Join(dataDir, "schemas.conf")
	settings.Enabled = true

	ioutil.WriteFile(settings.SchemasFile, []byte(SchemasOK), 0644)

	assert.NoError(settings.LoadAndValidate())
	return settings
}

func TestRespawn(t *testing.T) {
	assert := assert.New(t)

	// check all goroutine shutdowned
	helper.Root(t, func(root string) {
		startGoroutineNum := runtime.NumGoroutine()

		ch := points.NewChannel(0)
		settings := newTestSettings(t, path.Join(root, "init"))
		wsp := NewWhisper(ch, settings)

		for i := 0; i < 20; i++ {
			settings := newTestSettings(t, path.Join(root, fmt.Sprintf("%d", i)))
			if i%3 == 0 {
				settings.Enabled = false
			}
			wsp = Respawn(wsp, settings, ch)
		}

		wsp.Stop()

		time.Sleep(time.Second)

		// p := pprof.Lookup("goroutine")
		// p.WriteTo(os.Stdout, 1)

		endGoroutineNum := runtime.NumGoroutine()

		// GC worker etc
		assert.InEpsilon(startGoroutineNum, endGoroutineNum, 2)
	})
}
