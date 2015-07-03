package carbon

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Root(t *testing.T, callback func(dir string)) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatal(err)
		}
	}()

	callback(tmpDir)
}

func NewTestConfig(rootDir string) *Config {
	cfg := NewConfig()
	return cfg
}

func NewTestCarbon(config *Config) *Carbon {
	carbon := New()
	carbon.Configure(config)
	return carbon
}

func TestCarbon(t *testing.T) {
	assert := assert.New(t)
	Root(t, func(rootDir string) {
		config := NewTestConfig(rootDir)
		carbon := NewTestCarbon(config)

		assert.NotNil(carbon)
	})
}
