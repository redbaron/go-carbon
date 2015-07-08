package persister

import (
	"github.com/lomik/go-carbon/points"
	//"github.com/lomik/go-whisper"

	//"github.com/stretchr/testify/mock"

	"math/rand"
)

func randomPoints(num int, out chan *points.Points) {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	var i int
	for i = 0; i < num; i++ {
		b := make([]rune, 32)
		for i := range b {
			b[i] = letters[rand.Intn(len(letters))]
		}
		metric := string(b)
		p := points.OnePoint(metric, rand.Float64(), rand.Int63())
		out <- p
	}
}

// func TestShuffler(t *testing.T) {
// 	rand.Seed(time.Now().Unix())
// 	fixture := NewWhisper(in, settings)
// 	in := points.NewChannel(0)
// 	out1 := points.NewChannel(0)
// 	out2 := points.NewChannel(0)
// 	out3 := points.NewChannel(0)
// 	out4 := points.NewChannel(0)
// 	out := []*points.Channel{out1, out2, out3, out4}
// 	go fixture.shuffler(in, out)
// 	buckets := [4]int{0, 0, 0, 0}
// 	dotest := make(chan bool)
// 	runlength := 10000
// 	go func() {
// 		for {
// 			select {
// 			case <-out1.Chan():
// 				buckets[0]++
// 			case <-out2.Chan():
// 				buckets[1]++
// 			case <-out3.Chan():
// 				buckets[2]++
// 			case <-out4.Chan():
// 				buckets[3]++
// 			case <-dotest:
// 				total := 0
// 				for b := range buckets {
// 					assert.InEpsilon(t, float64(runlength)/4, buckets[b], (float64(runlength)/4)*.005, fmt.Sprintf("shuffle distribution is greater than .5% across 4 buckets after %d inputs", runlength))
// 					total += buckets[b]
// 				}
// 				assert.Equal(t, runlength, total, "total output of shuffle is not equal to input")

// 			}

// 		}
// 	}()
// 	randomPoints(runlength, in.Chan())
// 	fixture.exit <- true
// 	dotest <- true
// }

// func TestStop(t *testing.T) {
// 	fixture := Whisper{exit: make(chan bool)}
// 	timeout := make(chan bool, 1)
// 	go func() {
// 		time.Sleep(1 * time.Second)
// 		timeout <- true
// 	}()
// 	fixture.Stop()
// 	select {
// 	case _, ok := <-fixture.exit:
// 		assert.False(t, ok, "close caused a write to the exit channel")
// 		// a read from ch has occurred
// 	case _, ok := <-timeout:
// 		assert.False(t, ok, "close failed to close the exit channel in a reasonable time")
// 		// the read from ch has timed out
// 	}
// }
