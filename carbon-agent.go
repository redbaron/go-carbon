package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/user"
	"runtime"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/lomik/go-carbon/cache"
	"github.com/lomik/go-carbon/carbon"
	"github.com/lomik/go-carbon/config"
	"github.com/lomik/go-carbon/logging"
	"github.com/lomik/go-carbon/persister"
	"github.com/lomik/go-carbon/receiver"
)

import _ "net/http/pprof"

// Version of go-carbon
const Version = "0.5.1"

func main() {
	var err error

	/* CONFIG start */

	checkConfig := flag.Bool("check-config", false, "Check config and exit")

	printVersion := flag.Bool("version", false, "Print version")

	isDaemon := flag.Bool("daemon", false, "Run in background")
	pidfile := flag.String("pidfile", "", "Pidfile path (only for daemon)")

	flag.Parse()

	if *printVersion {
		fmt.Print(Version)
		return
	}

	cfg := carbon.NewConfig()

	if err = config.Parse(cfg); err != nil {
		log.Fatal(err)
	}

	var runAsUser *user.User
	if cfg.Common.User != "" {
		runAsUser, err = user.Lookup(cfg.Common.User)
		if err != nil {
			log.Fatal(err)
		}
	}

	var whisperSchemas *persister.WhisperSchemas
	var whisperAggregation *persister.WhisperAggregation

	if cfg.Whisper.Enabled {
		whisperSchemas, err = persister.ReadWhisperSchemas(cfg.Whisper.Schemas)
		if err != nil {
			log.Fatal(err)
		}

		if cfg.Whisper.Aggregation != "" {
			whisperAggregation, err = persister.ReadWhisperAggregation(cfg.Whisper.Aggregation)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			whisperAggregation = persister.NewWhisperAggregation()
		}
	}

	if err := logging.SetLevel(cfg.Common.LogLevel); err != nil {
		log.Fatal(err)
	}

	if *checkConfig {
		return
	}

	if err := logging.PrepareFile(cfg.Common.Logfile, runAsUser); err != nil {
		logrus.Fatal(err)
	}

	if err := logging.SetFile(cfg.Common.Logfile); err != nil {
		logrus.Fatal(err)
	}

	if *isDaemon {
		config.Daemonize(runAsUser, *pidfile)
	}

	logrus.SetLevel(logrus.DebugLevel)

	runtime.GOMAXPROCS(cfg.Common.MaxCPU)

	/* CONFIG end */

	// pprof
	if cfg.Pprof.Enabled {
		go func() {
			logrus.Fatal(http.ListenAndServe(cfg.Pprof.Listen, nil))
		}()
	}

	// carbon-cache prefix
	if hostname, err := os.Hostname(); err == nil {
		hostname = strings.Replace(hostname, ".", "_", -1)
		cfg.Common.GraphPrefix = strings.Replace(cfg.Common.GraphPrefix, "{host}", hostname, -1)
	} else {
		cfg.Common.GraphPrefix = strings.Replace(cfg.Common.GraphPrefix, "{host}", "localhost", -1)
	}

	core := cache.New()
	core.SetGraphPrefix(cfg.Common.GraphPrefix)
	core.SetMaxSize(cfg.Cache.MaxSize)
	core.SetInputCapacity(cfg.Cache.InputBuffer)
	core.Start()
	defer core.Stop()

	/* UDP start */
	udpCfg := cfg.Udp
	if udpCfg.Enabled {
		udpAddr, err := net.ResolveUDPAddr("udp", udpCfg.Listen)
		if err != nil {
			log.Fatal(err)
		}

		udpListener := receiver.NewUDP(core.In())
		udpListener.SetGraphPrefix(cfg.Common.GraphPrefix)

		if udpCfg.LogIncomplete {
			udpListener.SetLogIncomplete(true)
		}

		defer udpListener.Stop()
		if err = udpListener.Listen(udpAddr); err != nil {
			log.Fatal(err)
		}
	}
	/* UDP end */

	/* TCP start */
	tcpCfg := cfg.Tcp

	if tcpCfg.Enabled {
		tcpAddr, err := net.ResolveTCPAddr("tcp", tcpCfg.Listen)
		if err != nil {
			log.Fatal(err)
		}

		tcpListener := receiver.NewTCP(core.In())
		tcpListener.SetGraphPrefix(cfg.Common.GraphPrefix)

		defer tcpListener.Stop()
		if err = tcpListener.Listen(tcpAddr); err != nil {
			log.Fatal(err)
		}
	}
	/* TCP end */

	/* PICKLE start */
	pickleCfg := cfg.Pickle

	if pickleCfg.Enabled {
		pickleAddr, err := net.ResolveTCPAddr("tcp", pickleCfg.Listen)
		if err != nil {
			log.Fatal(err)
		}

		pickleListener := receiver.NewPickle(core.In())
		pickleListener.SetGraphPrefix(cfg.Common.GraphPrefix)

		defer pickleListener.Stop()
		if err = pickleListener.Listen(pickleAddr); err != nil {
			log.Fatal(err)
		}
	}
	/* PICKLE end */

	/* WHISPER start */
	if cfg.Whisper.Enabled {
		whisperPersister := persister.NewWhisper(cfg.Whisper.DataDir, whisperSchemas, whisperAggregation, core.Out())
		whisperPersister.SetGraphPrefix(cfg.Common.GraphPrefix)
		whisperPersister.SetMaxUpdatesPerSecond(cfg.Whisper.MaxUpdatesPerSecond)
		whisperPersister.SetWorkers(cfg.Whisper.Workers)

		whisperPersister.Start()
		defer whisperPersister.Stop()
	}
	/* WHISPER end */

	/* CARBONLINK start */
	if cfg.Carbonlink.Enabled {
		linkAddr, err := net.ResolveTCPAddr("tcp", cfg.Carbonlink.Listen)
		if err != nil {
			log.Fatal(err)
		}

		carbonlink := cache.NewCarbonlinkListener(core.Query())
		carbonlink.SetReadTimeout(cfg.Carbonlink.ReadTimeout.Value())
		carbonlink.SetQueryTimeout(cfg.Carbonlink.QueryTimeout.Value())

		defer carbonlink.Stop()
		if err = carbonlink.Listen(linkAddr); err != nil {
			log.Fatal(err)
		}

	}
	/* CARBONLINK end */

	logrus.Info("started")
	select {}
}
