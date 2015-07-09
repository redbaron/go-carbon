package carbon

import (
	"net"
	"testing"

	"github.com/lomik/go-carbon/helper"
	"github.com/lomik/go-carbon/points"
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

		for i := 0; i < 100; i++ {
			_, err := conn.Write(
				[]byte(
					points.NowPoint("metric1", float64(i)).String(),
				),
			)
			assert.NoError(err)
		}

	})
}
