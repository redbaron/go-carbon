package points

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestThrottleChan(t *testing.T) {
	perSecond := 100
	timestamp := time.Now().Unix()

	chIn := NewChannel(0)
	chOut := chIn.ThrottledOut(perSecond)
	wait := time.After(time.Second)

	bw := 0

	in, _ := chIn.Current()
	out, _ := chOut.Current()

loop:
	for {
		select {
		case <-wait:
			break loop
		default:
		}
		in <- OnePoint("metric", 1, timestamp)
		<-out
		bw++
	}
	close(in)

	max := float64(perSecond) * 1.05
	min := float64(perSecond) * 0.95

	assert.True(t, float64(bw) >= min)
	assert.True(t, float64(bw) <= max)
}
