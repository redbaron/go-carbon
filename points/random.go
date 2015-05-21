package points

import (
	"math/rand"
	"strings"
)

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
