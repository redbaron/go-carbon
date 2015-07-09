package points

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNextRandomMetric(t *testing.T) {
	assert := assert.New(t)

	rand.Seed(time.Now().Unix())

	times := 10000

	counts := []int{0, 0, 0}

	m := func(a float64, b int) int {
		return int(a * float64(b))
	}

	// start from empty
	for i := 0; i < times; i++ {
		counts[len(strings.Split(RandomNextName(""), "."))]++
	}
	assert.Equal(0, counts[0])
	assert.True(counts[1] > m(0.58, times))
	assert.True(counts[1] < m(0.62, times))

	counts = []int{0, 0, 0}

	// metric in root
	for i := 0; i < times; i++ {
		counts[len(strings.Split(RandomNextName("metricInRoot"), "."))]++
	}
	assert.Equal(0, counts[0])
	assert.True(counts[1] > m(0.58, times))
	assert.True(counts[1] < m(0.62, times))

	counts = []int{0, 0, 0, 0, 0}

	// metric in directory
	for i := 0; i < times; i++ {
		counts[len(strings.Split(RandomNextName("metric.In.Directory"), "."))]++
	}
	assert.Equal(0, counts[0])
	assert.Equal(0, counts[0])
	assert.True(counts[2] > m(0.28, times))
	assert.True(counts[2] < m(0.32, times))
	assert.True(counts[4] > m(0.18, times))
	assert.True(counts[4] < m(0.22, times))
}
