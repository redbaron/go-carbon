package persister

import (
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/lomik/go-carbon/points"
)

func withDirectory(t *testing.T, callback func(string)) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatal(err)
		}
	}()

	callback(dir)
}

func TestFull(t *testing.T) {
	// assert := assert.New(t)

	withDirectory(t, func(rootDir string) {

		inchan := make(chan *points.Points, 1024)
		schemas := WhisperSchemas{}
		aggrs := WhisperAggregation{}

		p := NewWhisper(rootDir, &schemas, &aggrs, inchan)

		p.SetGraphPrefix("carbon.agents.localhost.")
		p.SetWorkers(16)
		p.SetStatInterval(time.Hour) // do not autostat. run doCheckpoint manually

		rand.Seed(time.Now().Unix())
		points.RandomNames(1000)

	})

}
