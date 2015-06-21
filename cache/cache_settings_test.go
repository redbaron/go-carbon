package cache

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/lomik/go-carbon/logging"
	"github.com/stretchr/testify/assert"
)

func TestCacheSettings(t *testing.T) {
	assert := assert.New(t)

	testMaxSize := func(cache *Cache) {
		table := []int{0, 1, 100, 10000000}

		for _, value := range table {
			logging.Test(func(log *bytes.Buffer) {
				cache.EditSettings(func(settings *Settings) {
					settings.MaxSize = value
				})

				assert.Contains(log.String(), fmt.Sprintf("new=%#v", value))
			})

			assert.Equal(value, cache.settings.MaxSize)
		}
	}

	testGraphPrefix := func(cache *Cache) {
		table := []string{"carbon", "graphite"}

		for _, value := range table {
			logging.Test(func(log *bytes.Buffer) {
				cache.EditSettings(func(settings *Settings) {
					settings.GraphPrefix = value
				})

				assert.Contains(log.String(), fmt.Sprintf("new=%#v", value))
			})

			assert.Equal(value, cache.settings.GraphPrefix)
		}
	}

	testInputCapacity := func(cache *Cache) {
		table := []int{0, 1, 100, 10000000}

		for _, value := range table {
			logging.Test(func(log *bytes.Buffer) {
				cache.EditSettings(func(settings *Settings) {
					settings.InputCapacity = value
				})

				assert.Contains(log.String(), fmt.Sprintf("new=%#v", value))
			})

			assert.Equal(value, cache.settings.InputCapacity)
			assert.Equal(value, cap(cache.In().Chan()))
		}
	}

	testOutputCapacity := func(cache *Cache) {
		table := []int{0, 1, 100, 10000000}

		for _, value := range table {
			logging.Test(func(log *bytes.Buffer) {
				cache.EditSettings(func(settings *Settings) {
					settings.OutputCapacity = value
				})

				assert.Contains(log.String(), fmt.Sprintf("new=%#v", value))
			})

			assert.Equal(value, cache.settings.OutputCapacity)
			assert.Equal(value, cap(cache.Out().Chan()))
		}
	}

	// stopped and running
	for _, running := range []bool{false, true} {
		func() {
			cache := New()
			if running {
				cache.Start()
				defer cache.Stop()
			}

			testMaxSize(cache)
			testGraphPrefix(cache)
			testInputCapacity(cache)
			testOutputCapacity(cache)
		}()
	}
}
