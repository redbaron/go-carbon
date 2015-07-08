package persister

import (
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/lomik/go-whisper"

	"github.com/lomik/go-carbon/points"
)

// Whisper write data to *.wsp files
type Whisper struct {
	settings         *Settings
	in               *points.Channel
	exit             *Exit
	updateOperations uint32
	commitedPoints   uint32
	created          uint32 // counter
}

// NewWhisper create instance of Whisper
func NewWhisper(in *points.Channel, settings *Settings) *Whisper {
	return &Whisper{
		settings: settings,
		in:       in,
	}
}

func (p *Whisper) store(values *points.Points) {
	rootPath := p.settings.RootPath
	schemas := p.settings.schemas
	aggregation := p.settings.aggregation

	path := filepath.Join(rootPath, strings.Replace(values.Metric, ".", "/", -1)+".wsp")

	w, err := whisper.Open(path)
	if err != nil {
		schema := schemas.match(values.Metric)
		if schema == nil {
			logrus.Errorf("[persister] No storage schema defined for %s", values.Metric)
			return
		}

		aggr := aggregation.match(values.Metric)
		if aggr == nil {
			logrus.Errorf("[persister] No storage aggregation defined for %s", values.Metric)
			return
		}

		logrus.WithFields(logrus.Fields{
			"retention":    schema.retentionStr,
			"schema":       schema.name,
			"aggregation":  aggr.name,
			"xFilesFactor": aggr.xFilesFactor,
			"method":       aggr.aggregationMethodStr,
		}).Debugf("[persister] Creating %s", path)

		if err = os.MkdirAll(filepath.Dir(path), os.ModeDir|os.ModePerm); err != nil {
			logrus.Error(err)
			return
		}

		w, err = whisper.Create(path, schema.retentions, aggr.aggregationMethod, float32(aggr.xFilesFactor))
		if err != nil {
			logrus.Errorf("[persister] Failed to create new whisper file %s: %s", path, err.Error())
			return
		}

		atomic.AddUint32(&p.created, 1)
	}

	points := make([]*whisper.TimeSeriesPoint, len(values.Data))
	for i, r := range values.Data {
		points[i] = &whisper.TimeSeriesPoint{Time: int(r.Timestamp), Value: r.Value}
	}

	atomic.AddUint32(&p.commitedPoints, uint32(len(values.Data)))
	atomic.AddUint32(&p.updateOperations, 1)

	defer w.Close()

	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("[persister] UpdateMany %s recovered: %s", path, r)
		}
	}()
	w.UpdateMany(points)
}

func (p *Whisper) worker(inChannel *points.Channel, exit *Exit) {
	in, inChanged := inChannel.Out()

	for {
		select {
		// confirm exit and exit
		case <-exit.C:
			exit.Done()
			break
		// input channel resized
		case <-inChanged:
			in, inChanged = inChannel.Out()
		// store values
		case values := <-in:
			p.store(values)
		}
	}
}

func (p *Whisper) shuffler(inChannel *points.Channel, exit *Exit) {
	workersExit := NewExit(p.settings.Workers)

	var sendChannels [](chan<- *points.Points)
	var channels [](*points.Channel)

	for i := 0; i < p.settings.Workers; i++ {
		ch := points.NewChannel(32)
		channels = append(channels, ch)
		sendChannels = append(sendChannels, ch.InChan())
		go p.worker(ch, workersExit)
	}

	quarantine := func() { // receive all from channels
		out, outChanged := inChannel.In()
		for _, pointsChannel := range channels {
			ch := pointsChannel.OutChan()
			for {
				select {
				case <-outChanged:
					out, outChanged = inChannel.In()
				case p := <-ch:
					out <- p
				// all from ch readed
				default:
					close(pointsChannel.InChan())
					break
				}
			}
		}
	}

	workers := uint32(p.settings.Workers)
	in, inChanged := inChannel.Out()

	for {
		select {
		case <-inChanged:
			in, inChanged = inChannel.Out()
		case values := <-in:
			index := crc32.ChecksumIEEE([]byte(values.Metric)) % workers
			sendChannels[index] <- values
		case <-exit.C:
			workersExit.Exit() // wait for all workers stopped
			exit.Done()
			go quarantine()
		}
	}
}

// save stat
func (p *Whisper) doCheckpoint() {
	graphPrefix := p.settings.GraphPrefix

	statChan := p.in.InChan()

	// Stat sends internal statistics to cache
	stat := func(metric string, value float64) {
		statChan <- points.NowPoint(fmt.Sprintf("%spersister.%s", graphPrefix, metric), value)
	}

	load := func(addr *uint32) uint32 {
		value := atomic.LoadUint32(addr)
		atomic.AddUint32(addr, -value)
		return value
	}

	updateOperations := load(&p.updateOperations)
	commitedPoints := load(&p.commitedPoints)
	created := load(&p.created)

	logrus.WithFields(logrus.Fields{
		"updateOperations": float64(updateOperations),
		"commitedPoints":   float64(commitedPoints),
		"created":          created,
	}).Info("[persister] doCheckpoint()")

	stat("updateOperations", float64(updateOperations))
	stat("commitedPoints", float64(commitedPoints))
	if updateOperations > 0 {
		stat("pointsPerUpdate", float64(commitedPoints)/float64(updateOperations))
	} else {
		stat("pointsPerUpdate", 0.0)
	}

	stat("created", float64(created))
}

// stat timer
func (p *Whisper) statWorker(exit chan bool) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-exit:
			break
		case <-ticker.C:
			go p.doCheckpoint()
		}
	}
}

// Start worker
func (p *Whisper) Start() {
	exit := NewExit(1)

	p.exit = exit

	go p.statWorker(exit.C)

	inChan := p.in
	inChan.Throttle(p.settings.MaxUpdatesPerSecond)

	if p.settings.Workers <= 1 { // solo worker
		go p.worker(inChan, exit)
	} else {
		go p.shuffler(inChan, exit)
	}
}

// Stop worker
func (p *Whisper) Stop() {
	p.exit.Exit()
}
