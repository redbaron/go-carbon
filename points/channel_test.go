package points

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestChannel(t *testing.T) {
	assert := assert.New(t)

	c := NewChannel(10)
	c.closeOldTimeout = 50 * time.Millisecond

	out, _ := c.Current()
	go func() {
		for i := 0; i < 100; i++ {
			out <- NowPoint("test", float64(i))
		}
	}()

	var result []*Points

	in, _ := c.Current()
	assert.Equal(10, cap(in))

	for j := 0; j < 50; j++ {
		p := <-in
		result = append(result, p)
	}

	c.Resize(20)

	in, _ = c.Current()
	assert.Equal(20, cap(in))

	for j := 0; j < 50; j++ {
		p := <-in
		result = append(result, p)
	}

	assert.Equal(99.0, result[99].Data[0].Value)

	// check not closed original
	select {
	case <-out:
		assert.Fail("original channel is closed")
	default:
	}

	// check closed after timeout
	time.Sleep(200 * time.Millisecond)
	select {
	case _, opened := <-out:
		assert.False(opened)
	default:
		assert.Fail("original channel is not closed")
	}
}
