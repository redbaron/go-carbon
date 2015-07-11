package carbon

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/lomik/go-carbon/helper"
	"github.com/stretchr/testify/assert"
)

func NewTestConfig(rootDir string) *Config {
	cfg := NewConfig()

	cfg.Common.Logfile = filepath.Join(rootDir, "go-carbon.log")

	cfg.Whisper.DataDir = rootDir
	cfg.Whisper.Schemas = filepath.Join(rootDir, "schemas.conf")

	return cfg
}

func NewTestCarbon(config *Config) (*Carbon, error) {
	carbon := New()
	err := carbon.Configure(config, true)
	return carbon, err
}

var SchemasOK = `
[default]
pattern = .*
retentions = 1s:10m
`

func writeSchemas(config *Config, content string) {
	ioutil.WriteFile(config.Whisper.Schemas, []byte(content), 0644)
}

func TestCarbon(t *testing.T) {
	assert := assert.New(t)
	helper.Root(t, func(rootDir string) {

		config := NewTestConfig(rootDir)
		writeSchemas(config, SchemasOK)
		app, err := NewTestCarbon(config)

		assert.NoError(err)
		assert.NotNil(app)

	})
}
