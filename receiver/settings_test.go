package receiver

import (
	"testing"

	"github.com/lomik/go-carbon/points"
	"github.com/stretchr/testify/assert"
)

type settingsTestCase struct {
	rcvTypes   int
	changeFunc func(*Settings)
	validator  func(*Receiver)
}

type rcvTestFactory struct {
	setUp    func(*points.Channel) *Receiver
	tearDown func(*Receiver)
	validate func(*Receiver)
}

func TestSettingsChange(t *testing.T) {
	assert := assert.New(t)

	// testTCP := 1 << uint(typeTCP)
	// testUDP := 1 << uint(typeUDP)
	// testPickle := 1 << uint(typePickle)

	// table := []*settingsTestCase{
	// 	&settingsTestCase{
	// 		testTCP | testPickle,
	// 	},
	// }

	hasMetric := func(name string, ch *points.Channel) bool {
		exists := false
	LOOP:
		for {
			select {
			case p := <-ch.OutChan():
				if p.Metric == name {
					exists = true
				}
			default:
				break LOOP
			}
		}
		return exists
	}

	receivers := []*rcvTestFactory{
		// UDP
		&rcvTestFactory{
			setUp: func(out *points.Channel) *Receiver {
				rcv := NewUDP(out)
				s := rcv.Settings()

				s.Enabled = true
				s.GraphPrefix = "init_graph_prefix."
				s.ListenAddr = "127.0.0.1:22003"
				s.LogIncomplete = false

				assert.NoError(s.Apply())

				rcv.start()
				return rcv
			},
			tearDown: func(rcv *Receiver) {
				rcv.Stop()
			},
			validate: func(rcv *Receiver) {
				assert.NotNil(rcv.Addr())
			},
		},
	}

	for _, factory := range receivers {
		ch := points.NewChannel(1024)
		rcv := factory.setUp(ch)
		factory.validate(rcv)
		factory.tearDown(rcv)
	}
}
