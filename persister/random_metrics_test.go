package persister

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func randomString(stringLen int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	b := make([]rune, stringLen)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}

	return string(b)
}

func nextRandomMetric(prev string) string {
	/*
		prev in root directory. next:
		   60% in same directory
		   40% in subdirectory

		prev not in root directory. next:
		   50% in same directory
		   30% in parent directory
		   20% in subdirectory
	*/

	// string length = base + random
	stringBaseLength := 8
	stringRandomLength := 10

	randomName := func() string {
		return randomString(stringBaseLength + rand.Intn(stringRandomLength))
	}

	s := strings.Split(prev, ".")
	level := len(s)

	dir := s[:len(s)-1]

	rnd := rand.Intn(100)

	if level == 1 { // root
		if rnd >= 60 {
			// sub
			dir = append(dir, randomName())
		}
	} else {
		if rnd >= 80 {
			// sub
			dir = append(dir, randomName())
		} else if rnd >= 50 {
			// parent
			dir = dir[:len(dir)-1]
		}
	}

	metric := append(dir, randomName())

	return strings.Join(metric, ".")
}

func generateMetricNames(count int) []string {
	result := []string{}
	for i := 0; i < count; i++ {
		if i == 0 {
			result = append(result, nextRandomMetric(""))
		} else {
			result = append(result, nextRandomMetric(result[i-1]))
		}
	}

	return result
}

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
		counts[len(strings.Split(nextRandomMetric(""), "."))]++
	}
	assert.Equal(0, counts[0])
	assert.True(counts[1] > m(0.58, times))
	assert.True(counts[1] < m(0.62, times))

	counts = []int{0, 0, 0}

	// metric in root
	for i := 0; i < times; i++ {
		counts[len(strings.Split(nextRandomMetric("metricInRoot"), "."))]++
	}
	assert.Equal(0, counts[0])
	assert.True(counts[1] > m(0.58, times))
	assert.True(counts[1] < m(0.62, times))

	counts = []int{0, 0, 0, 0, 0}

	// metric in directory
	for i := 0; i < times; i++ {
		counts[len(strings.Split(nextRandomMetric("metric.In.Directory"), "."))]++
	}
	assert.Equal(0, counts[0])
	assert.Equal(0, counts[0])
	assert.True(counts[2] > m(0.28, times))
	assert.True(counts[2] < m(0.32, times))
	assert.True(counts[4] > m(0.18, times))
	assert.True(counts[4] < m(0.22, times))
}
