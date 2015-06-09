package points

import (
	"math/rand"
	"strings"
	"time"
)

// RandomPoints array generator. Configure with chain setters and execute Make()
type RandomPoints struct {
	metric             string
	minValue           float64
	maxValue           float64
	minPointsPerUpdate int
	maxPointsPerUpdate int
}

// NewRandomPoints returns instant of RandomPoints
func NewRandomPoints() *RandomPoints {
	return &RandomPoints{
		metric:             "",
		minValue:           0.0,
		maxValue:           1024.0,
		minPointsPerUpdate: 1,
		maxPointsPerUpdate: 60,
	}
}

// Copy RandomPoints instance to another
func (r *RandomPoints) Copy() *RandomPoints {
	return &RandomPoints{
		metric:             r.metric,
		minValue:           r.minValue,
		maxValue:           r.maxValue,
		minPointsPerUpdate: r.minPointsPerUpdate,
		maxPointsPerUpdate: r.maxPointsPerUpdate,
	}
}

// WithMetric ...
func (r *RandomPoints) WithMetric(metric string) *RandomPoints {
	c := r.Copy()
	c.metric = metric
	return c
}

// WithMinValue ...
func (r *RandomPoints) WithMinValue(minValue float64) *RandomPoints {
	c := r.Copy()
	c.minValue = minValue
	return c
}

// WithMaxValue ...
func (r *RandomPoints) WithMaxValue(maxValue float64) *RandomPoints {
	c := r.Copy()
	c.maxValue = maxValue
	return c
}

// WithMinPointsPerUpdate ...
func (r *RandomPoints) WithMinPointsPerUpdate(minPointsPerUpdate int) *RandomPoints {
	c := r.Copy()
	c.minPointsPerUpdate = minPointsPerUpdate
	return c
}

// WithMaxPointsPerUpdate ...
func (r *RandomPoints) WithMaxPointsPerUpdate(maxPointsPerUpdate int) *RandomPoints {
	c := r.Copy()
	c.maxPointsPerUpdate = maxPointsPerUpdate
	return c
}

// RandomString generates part of metric name
func RandomString(stringLen int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_")

	b := make([]rune, stringLen)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}

	return string(b)
}

// RandomNextName generates new metric name in same/parent/sub directory
//     prev in root directory. next:
//         60% in same directory
//         40% in subdirectory
//     prev not in root directory. next:
//         50% in same directory
//         30% in parent directory
//         20% in subdirectory
func RandomNextName(prev string) string {
	// string length = base + random
	stringBaseLength := 8
	stringRandomLength := 10

	randomName := func() string {
		return RandomString(stringBaseLength + rand.Intn(stringRandomLength))
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

// RandomNames generates list of metric names
func RandomNames(count int) []string {
	rand.Seed(time.Now().Unix())

	result := []string{}
	for i := 0; i < count; i++ {
		if i == 0 {
			result = append(result, RandomNextName(""))
		} else {
			result = append(result, RandomNextName(result[i-1]))
		}
	}

	return result
}
