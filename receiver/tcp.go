package receiver

import (
	"bufio"
	"encoding/binary"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/lomik/go-carbon/points"

	"github.com/Sirupsen/logrus"
)

// NewTCP create new instance of TCP Receiver
func NewTCP(out *points.Channel) *Receiver {
	rcv := new(out)
	rcv.rcvType = typeTCP
	return rcv
}

// NewPickle create new instance of Receiver with pickle listener enabled
func NewPickle(out *points.Channel) *Receiver {
	rcv := new(out)
	rcv.rcvType = typePICKLE
	return rcv
}

func (rcv *Receiver) handleTCP(conn net.Conn) {
	atomic.AddInt32(&rcv.active, 1)
	defer atomic.AddInt32(&rcv.active, -1)

	defer conn.Close()
	reader := bufio.NewReader(conn)

	out, outChanged := rcv.out.Current()

	for {
		conn.SetReadDeadline(time.Now().Add(2 * time.Minute))

		line, err := reader.ReadBytes('\n')

		if err != nil {
			if err == io.EOF {
				if len(line) > 0 {
					logrus.Warningf("[%s] Unfinished line: %#v", rcv.TypeString(), line)
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

func (rcv *Receiver) handlePickle(conn net.Conn) {
	atomic.AddInt32(&rcv.active, 1)
	defer atomic.AddInt32(&rcv.active, -1)

	defer conn.Close()
	reader := bufio.NewReader(conn)

	var msgLen uint32
	var err error

	out, outChanged := rcv.out.Current()

	for {
		conn.SetReadDeadline(time.Now().Add(2 * time.Minute))

		// Read prepended length
		err = binary.Read(reader, binary.BigEndian, &msgLen)
		if err != nil {
			if err == io.EOF {
				return
			}

			atomic.AddUint32(&rcv.errors, 1)
			logrus.Warningf("[%s] Can't read message length: %s", rcv.TypeString(), err.Error())
			return
		}

		// Allocate a byte array of the expected length
		data := make([]byte, msgLen)

		// Read remainder of pickle packet into byte array
		if err = binary.Read(reader, binary.BigEndian, data); err != nil {
			atomic.AddUint32(&rcv.errors, 1)
			logrus.Warningf("[%s] Can't read message body: %s", rcv.TypeString(), err.Error())
			return
		}

		msgs, err := points.ParsePickle(data)

		if err != nil {
			atomic.AddUint32(&rcv.errors, 1)
			logrus.Infof("[%s] Can't unpickle message: %s", rcv.TypeString(), err.Error())
			return
		}

		for _, msg := range msgs {
			atomic.AddUint32(&rcv.metricsReceived, uint32(len(msg.Data)))

			select {
			case <-outChanged:
				out, outChanged = rcv.out.Current()
			default:
			}
			out <- msg
		}
	}
}

// ListenTCP bind port. Receive messages and send to out channel
func (rcv *Receiver) ListenTCP(addr *net.TCPAddr) error {

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}

	rcv.addr = listener.Addr()

	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				rcv.doCheckpoint()
			case <-rcv.exit:
				listener.Close()
				return
			}
		}
	}()

	handler := rcv.handleTCP
	if rcv.rcvType == typePICKLE {
		handler = rcv.handlePickle
	}

	go func() {
		defer listener.Close()

		for {

			conn, err := listener.Accept()
			if err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					break
				}
				logrus.Warningf("[%s] Failed to accept connection: %s", rcv.TypeString(), err)
				continue
			}

			go handler(conn)
		}

	}()

	return nil
}
