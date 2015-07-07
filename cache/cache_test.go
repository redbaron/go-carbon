package cache

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/lomik/go-carbon/points"
	"github.com/stretchr/testify/assert"
)

func TestCache(t *testing.T) {
	assert := assert.New(t)

	c := New()

	c.add(points.OnePoint("hello.world", 42, 10))

	assert.Equal(1, c.size)

	c.add(points.OnePoint("hello.world", 15, 12))

	assert.Equal(2, c.size)

	values := c.pop()

	assert.Equal("hello.world", values.Metric)
	assert.Equal(2, len(values.Data))
	assert.Equal(0, c.size)
}

func TestCacheCheckpoint(t *testing.T) {
	assert := assert.New(t)

	cache := New()

	settings := cache.Settings()
	settings.OutputCapacity = 0
	assert.NoError(settings.Apply())

	cache.Start()
	defer cache.Stop()

	startTime := time.Now().Unix() - 60*60

	sizes := []int{1, 15, 42, 56, 22, 90, 1}

	for index, value := range sizes {
		metricName := fmt.Sprintf("metric%d", index)

		for i := value; i > 0; i-- {
			cache.In().InChan() <- points.OnePoint(metricName, float64(i), startTime+int64(i))
		}

	}

	time.Sleep(100 * time.Millisecond)
	// @todo: test log
	cache.doCheckpoint()

	d := <-cache.Out().OutChan()
	assert.Equal("metric0", d.Metric)

	systemMetrics := []string{
		"carbon.cache.inputLenAfterCheckpoint",
		"carbon.cache.inputLenBeforeCheckpoint",
		"carbon.cache.checkpointTime",
		"carbon.cache.overflow",
		"carbon.cache.queries",
		"carbon.cache.metrics",
		"carbon.cache.size",
	}

	for _, metricName := range systemMetrics {
		d = <-cache.Out().OutChan()
		assert.Equal(metricName, d.Metric)
	}

	result := sizes[1:]
	sort.Sort(sort.Reverse(sort.IntSlice(result)))

	for _, size := range result {
		d = <-cache.Out().OutChan()
		assert.Equal(size, len(d.Data))
	}
}
