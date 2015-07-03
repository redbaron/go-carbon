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

	// parse file, print default config, check config
	if err = config.Parse(cfg); err != nil {
		log.Fatal(err)
	}

	// parse schemas, aggregation
	if err = cfg.Load(); err != nil {
		log.Fatal(err)
	}

	var runAsUser *user.User
	if cfg.Common.User != "" {
		runAsUser, err = user.Lookup(cfg.Common.User)
		if err != nil {
			log.Fatal(err)
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

	app := carbon.New()
	if err := app.Configure(cfg); err != nil {
		logrus.Fatal(err)
		return
	}

	logrus.Info("go-carbon started")
	select {}

	// carbon-cache prefix
	if hostname, err := os.Hostname(); err == nil {
		hostname = strings.Replace(hostname, ".", "_", -1)
		cfg.Common.GraphPrefix = strings.Replace(cfg.Common.GraphPrefix, "{host}", hostname, -1)
	} else {
		cfg.Common.GraphPrefix = strings.Replace(cfg.Common.GraphPrefix, "{host}", "localhost", -1)
	}

	core := cache.New()
	coreSettings := core.Settings()
	coreSettings.GraphPrefix = cfg.Common.GraphPrefix
	coreSettings.MaxSize = cfg.Cache.MaxSize
	coreSettings.InputCapacity = cfg.Cache.InputBuffer
	if err := coreSettings.Apply(); err != nil {
		logrus.Fatal(err)
		return
	}
	core.Start()
	defer core.Stop()

	/* UDP start */
	udpCfg := cfg.UDP
	if udpCfg.Enabled {
		udpAddr, err := net.ResolveUDPAddr("udp", udpCfg.Listen)
		if err != nil {
			log.Fatal(err)
		}

		udpListener := receiver.NewUDP(core.In())

		udpSettings := udpListener.Settings()
		udpSettings.GraphPrefix = cfg.Common.GraphPrefix
		udpSettings.LogIncomplete = udpCfg.LogIncomplete
		udpSettings.Apply()

		defer udpListener.Stop()
		if err = udpListener.ListenUDP(udpAddr); err != nil {
			log.Fatal(err)
		}
	}
	/* UDP end */

	/* TCP start */
	tcpCfg := cfg.TCP

	if tcpCfg.Enabled {
		tcpAddr, err := net.ResolveTCPAddr("tcp", tcpCfg.Listen)
		if err != nil {
			log.Fatal(err)
		}

		tcpListener := receiver.NewTCP(core.In())

		tcpSettings := tcpListener.Settings()
		tcpSettings.GraphPrefix = cfg.Common.GraphPrefix
		tcpSettings.Apply()

		defer tcpListener.Stop()
		if err = tcpListener.ListenTCP(tcpAddr); err != nil {
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

		pickleSettings := pickleListener.Settings()
		pickleSettings.GraphPrefix = cfg.Common.GraphPrefix
		pickleSettings.Apply()

		defer pickleListener.Stop()
		if err = pickleListener.ListenTCP(pickleAddr); err != nil {
			log.Fatal(err)
		}
	}
	/* PICKLE end */

	/* WHISPER start */
	if cfg.Whisper.Enabled {
		whisperPersister := persister.NewWhisper(cfg.Whisper.DataDir, cfg.WhisperSchemas, cfg.WhisperAggregation, core.Out())
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
