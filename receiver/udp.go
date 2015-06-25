package receiver

import (
	"bytes"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/lomik/go-carbon/points"

	"github.com/Sirupsen/logrus"
)

// NewUDP create new instance of UDP
func NewUDP(out *points.Channel) *Receiver {
	rcv := new(out)
	rcv.rcvType = typeUDP
	return rcv
}

type incompleteRecord struct {
	deadline time.Time
	data     []byte
}

// incompleteStorage store incomplete lines
type incompleteStorage struct {
	Records   map[string]*incompleteRecord
	Expires   time.Duration
	NextPurge time.Time
	MaxSize   int
}

func newIncompleteStorage() *incompleteStorage {
	return &incompleteStorage{
		Records:   make(map[string]*incompleteRecord, 0),
		Expires:   5 * time.Second,
		MaxSize:   10000,
		NextPurge: time.Now().Add(time.Second),
	}
}

func (storage *incompleteStorage) store(addr string, data []byte) {
	storage.Records[addr] = &incompleteRecord{
		deadline: time.Now().Add(storage.Expires),
		data:     data,
	}
	storage.checkAndClear()
}

func (storage *incompleteStorage) pop(addr string) []byte {
	if record, ok := storage.Records[addr]; ok {
		delete(storage.Records, addr)
		if record.deadline.Before(time.Now()) {
			return nil
		}
		return record.data
	}
	return nil
}

func (storage *incompleteStorage) purge() {
	now := time.Now()
	for key, record := range storage.Records {
		if record.deadline.Before(now) {
			delete(storage.Records, key)
		}
	}
	storage.NextPurge = time.Now().Add(time.Second)
}

func (storage *incompleteStorage) checkAndClear() {
	if len(storage.Records) < storage.MaxSize {
		return
	}
	if storage.NextPurge.After(time.Now()) {
		return
	}
	storage.purge()
}

func logIncomplete(peer *net.UDPAddr, message []byte, lastLine []byte) {
	p1 := bytes.IndexByte(message, 0xa) // find first "\n"

	if p1 != -1 && p1+len(lastLine) < len(message)-10 { // print short version
		logrus.Warningf(
			"[udp] incomplete message from %s: \"%s\\n...(%d bytes)...\\n%s\"",
			peer.String(),
			string(message[:p1]),
			len(message)-p1-len(lastLine)-2,
			string(lastLine),
		)
	} else { // print full
		logrus.Warningf(
			"[udp] incomplete message from %s: %#v",
			peer.String(),
			string(message),
		)
	}
}

// ListenUDP bind port. Receive messages and send to out channel
func (rcv *Receiver) ListenUDP(addr *net.UDPAddr) error {
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}

	rcv.addr = conn.LocalAddr()

	go rcv.checkpointWorker()

	isLogIncomplete := rcv.settings.LogIncomplete

	go func() {
		var settingsChanged chan bool

		refreshSettings := func() {
			rcv.RLock()
			defer rcv.RUnlock()

			settingsChanged = rcv.settingsChanged
			isLogIncomplete = rcv.settings.LogIncomplete
		}

		refreshSettings()

		for {
			select {
			// settings updated
			case <-settingsChanged:
				refreshSettings()

			case <-rcv.exit:
				conn.Close()
				return
			}
		}
	}()

	go func() {
		defer conn.Close()

		var buf [2048]byte

		var data *bytes.Buffer

		lines := newIncompleteStorage()

		out, outChanged := rcv.out.Current()

		for {
			rlen, peer, err := conn.ReadFromUDP(buf[:])
			if err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					break
				}
				atomic.AddUint32(&rcv.errors, 1)
				logrus.Error(err)
				continue
			}

			prev := lines.pop(peer.String())

			if prev != nil {
				data = bytes.NewBuffer(prev)
				data.Write(buf[:rlen])
			} else {
				data = bytes.NewBuffer(buf[:rlen])
			}

			for {
				line, err := data.ReadBytes('\n')

				if err != nil {
					if err == io.EOF {
						if len(line) > 0 { // incomplete line received

							if isLogIncomplete {
								logIncomplete(peer, buf[:rlen], line)
							}

							lines.store(peer.String(), line)
							atomic.AddUint32(&rcv.incompleteReceived, 1)
						}
					} else {
						atomic.AddUint32(&rcv.errors, 1)
						logrus.Error(err)
					}
					break
				}
				if len(line) > 0 { // skip empty lines
					if msg, err := points.ParseText(string(line)); err != nil {
						atomic.AddUint32(&rcv.errors, 1)
						logrus.Info(err)
					} else {
						atomic.AddUint32(&rcv.metricsReceived, 1)

						select {
						case <-outChanged:
							out, outChanged = rcv.out.Current()
						default:
						}
						out <- msg
					}
				}
			}
		}

	}()

	return nil
}
