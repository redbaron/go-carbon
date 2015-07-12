package carbon

import (
	"fmt"
	"net"
	"path"
	"testing"
	"time"

	"github.com/lomik/go-carbon/helper"
	"github.com/lomik/go-carbon/points"
	"github.com/lomik/go-whisper"
	"github.com/stretchr/testify/assert"
)

func checkPersisted(t *testing.T, successExpected bool, app *Carbon, rootDir string, sendFunction func(p *points.Points)) {
	assert := assert.New(t)
	pointsCount := 100
	metricName := points.RandomString(10)

	ts := int64(time.Now().Unix())

	for i := 0; i < pointsCount; i++ {
		sendFunction(points.OnePoint(metricName, float64(i), ts-int64(i)))
	}

	time.Sleep(time.Second)

	if successExpected {
		wsp, err := whisper.Open(path.Join(rootDir, fmt.Sprintf("%s.wsp", metricName)))
		assert.NoError(err)

		savedData, err := wsp.Fetch(int(ts)-pointsCount, int(ts))
		assert.NoError(err)

		assert.Equal(pointsCount, len(savedData.Values()))

		for i, v := range savedData.Values() {
			assert.Equal(pointsCount-1-i, v)
		}
	} else {
		wsp, err := whisper.Open(path.Join(rootDir, fmt.Sprintf("%s.wsp", metricName)))
		assert.Error(err)
		assert.Nil(wsp)

		ch := app.Cache.Out().OutChan()
		cnt := 0
		for {
			msg := <-ch
			assert.Equal(metricName, msg.Metric)
			for _, p := range msg.Data {
				assert.Equal(cnt, p.Value)
				cnt++
			}
			if cnt == pointsCount {
				break
			}
		}
	}
}

func TestToggleWhisper(t *testing.T) {
	assert := assert.New(t)

	helper.Root(t, func(rootDir string) {

		config := NewTestConfig(rootDir)
		writeSchemas(config, SchemasOK)
		app, err := NewTestCarbon(config)

		assert.NoError(err)
		assert.NotNil(app)

		conn, err := net.Dial("tcp", app.TCP.Addr().String())
		assert.NoError(err)

		checkPersisted(t, true, app, rootDir, func(p *points.Points) {
			_, err := conn.Write([]byte(p.String()))
			assert.NoError(err)
		})

		conn.Close()

		// disable persister and try again
		config.Whisper.Enabled = false
		err = app.Configure(config, true)
		assert.NoError(err)

		conn, err = net.Dial("tcp", app.TCP.Addr().String())
		assert.NoError(err)

		checkPersisted(t, false, app, rootDir, func(p *points.Points) {
			_, err := conn.Write([]byte(p.String()))
			assert.NoError(err)
		})

		conn.Close()

		// enable persister
		config.Whisper.Enabled = true
		err = app.Configure(config, true)
		assert.NoError(err)

		conn, err = net.Dial("tcp", app.TCP.Addr().String())
		assert.NoError(err)

		checkPersisted(t, true, app, rootDir, func(p *points.Points) {
			_, err := conn.Write([]byte(p.String()))
			assert.NoError(err)
		})

		conn.Close()

	})
}
