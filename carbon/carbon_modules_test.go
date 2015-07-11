package carbon

import (
	"net"
	"path"
	"testing"
	"time"

	"github.com/lomik/go-carbon/helper"
	"github.com/lomik/go-carbon/points"
	"github.com/lomik/go-whisper"
	"github.com/stretchr/testify/assert"
)

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

		ts := int64(time.Now().Unix())

		pointsCount := 100

		for i := 0; i < pointsCount; i++ {
			_, err := conn.Write(
				[]byte(
					points.OnePoint("metric1", float64(i), ts-int64(i)).String(),
				),
			)
			assert.NoError(err)
		}

		time.Sleep(time.Second)

		wsp, err := whisper.Open(path.Join(rootDir, "metric1.wsp"))
		assert.NoError(err)

		savedData, err := wsp.Fetch(int(ts)-pointsCount, int(ts))
		assert.NoError(err)

		assert.Equal(pointsCount, len(savedData.Values()))

		for i, v := range savedData.Values() {
			assert.Equal(pointsCount-1-i, v)
		}

	})
}
