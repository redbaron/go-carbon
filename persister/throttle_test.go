package persister

import (
	"os"
	"testing"
)

func doTestThrottleChan(t *testing.T, perSecond int) {
}

func TestThrottleChan(t *testing.T) {
	perSecondTable := []int{1, 10, 100, 1000, 10000, 100000, 213000}

	if os.Getenv("TRAVIS") != "true" {
		perSecondTable = append(perSecondTable, 531234)
	}

	for _, perSecond := range perSecondTable {
		doTestThrottleChan(t, perSecond)
	}
}
