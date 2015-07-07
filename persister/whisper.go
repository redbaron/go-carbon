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
	exit             chan bool
	updateOperations uint32
	commitedPoints   uint32
	created          uint32 // counter
}

// NewWhisper create instance of Whisper
func NewWhisper(in *points.Channel) *Whisper {
	settings := &Settings{
		changed:     make(chan bool),
		Enabled:     false,
		GraphPrefix: "carbon.",
		Workers:     1,
	}
	p := &Whisper{
		settings: settings,
		in:       in,
		exit:     make(chan bool),
	}
	settings.persister = p
	return p
}

func (p *Whisper) store(values *points.Points, rootPath string, schemas *WhisperSchemas, aggregation *WhisperAggregation) {
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

func (p *Whisper) worker(inChannel *points.Channel) {
	in, inChanged := inChannel.Current()

	var rootPath string
	var schemas *WhisperSchemas
	var aggregation *WhisperAggregation
	var settingsChanged chan bool

	refreshSettings := func() {
		p.settings.RLock()
		defer p.settings.RUnlock()

		rootPath = p.settings.RootPath
		schemas = p.settings.schemas
		aggregation = p.settings.aggregation
	}

	refreshSettings()

	for {
		select {
		// @TODO: close in shuffler or loose cache
		case <-p.exit:
			break
		// input channel resized
		case <-inChanged:
			in, inChanged = inChannel.Current()
		// settings updated
		case <-settingsChanged:
			refreshSettings()
		case values := <-in:
			p.store(values, rootPath, schemas, aggregation)
		}
	}
}

func (p *Whisper) shuffler(inChannel *points.Channel, out []*points.Channel) {
	workers := uint32(len(out))

	var outChannels [](chan *points.Points)

	for _, c := range out {
		ch, _ := c.Current() // channels between shuffler and persister unchangeable
		outChannels = append(outChannels, ch)
	}

	in, inChanged := inChannel.Current()

	for {
		select {
		case <-p.exit:
			break
		case <-inChanged:
			in, inChanged = inChannel.Current()
		case values := <-in:
			index := crc32.ChecksumIEEE([]byte(values.Metric)) % workers
			outChannels[index] <- values
		}
	}
}

// save stat
func (p *Whisper) doCheckpoint() {

	p.settings.RLock()
	graphPrefix := p.settings.GraphPrefix
	p.settings.RUnlock()

	statChan := p.in.Chan()

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
func (p *Whisper) statWorker() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-p.exit:
			break
		case <-ticker.C:
			go p.doCheckpoint()
		}
	}
}

// Start worker
func (p *Whisper) Start() {
	go p.statWorker()

	inChan := p.in
	if p.maxUpdatesPerSecond > 0 {
		inChan = inChan.ThrottledOut(p.maxUpdatesPerSecond)
	}

	if p.workersCount <= 1 { // solo worker
		go p.worker(inChan)
	} else {
		var channels [](*points.Channel)

		for i := 0; i < p.workersCount; i++ {
			ch := points.NewChannel(32)
			channels = append(channels, ch)
			go p.worker(ch)
		}

		go p.shuffler(inChan, channels)
	}
}

// Stop worker
func (p *Whisper) Stop() {
	close(p.exit)
}
