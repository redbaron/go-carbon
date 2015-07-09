package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/user"
	"runtime"

	"github.com/Sirupsen/logrus"
	"github.com/lomik/go-carbon/carbon"
	"github.com/lomik/go-carbon/config"
	"github.com/lomik/go-carbon/logging"
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

	app := carbon.New()

	// only validate config
	if err = app.Configure(cfg, true); err != nil {
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

	// logrus.SetLevel(logrus.DebugLevel)

	runtime.GOMAXPROCS(cfg.Common.MaxCPU)

	/* CONFIG end */

	// pprof
	if cfg.Pprof.Enabled {
		go func() {
			logrus.Fatal(http.ListenAndServe(cfg.Pprof.Listen, nil))
		}()
	}

	// validate and APPLY settings
	if err := app.Configure(cfg, false); err != nil {
		logrus.Fatal(err)
		return
	}

	logrus.Info("go-carbon started")
	select {}
}
