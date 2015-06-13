package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/Sirupsen/logrus"
	"github.com/lomik/go-carbon/cache"
	"github.com/lomik/go-carbon/carbon"
	"github.com/lomik/go-carbon/logging"
	"github.com/lomik/go-carbon/persister"
	"github.com/lomik/go-carbon/receiver"
	"github.com/lomik/go-daemon"
)

import _ "net/http/pprof"

// Version of go-carbon
const Version = "0.5.1"

// PrintConfig ...
func PrintConfig(cfg interface{}) error {
	buf := new(bytes.Buffer)

	encoder := toml.NewEncoder(buf)
	encoder.Indent = ""

	if err := encoder.Encode(cfg); err != nil {
		return err
	}

	fmt.Print(buf.String())
	return nil
}

// ParseConfig ...
func ParseConfig(filename string, cfg interface{}) error {
	if filename != "" {
		if _, err := toml.DecodeFile(filename, cfg); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	var err error

	/* CONFIG start */

	configFile := flag.String("config", "", "Filename of config")
	printDefaultConfig := flag.Bool("config-print-default", false, "Print default config")
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

	if *printDefaultConfig {
		if err = PrintConfig(cfg); err != nil {
			log.Fatal(err)
		}
		return
	}

	if err = ParseConfig(*configFile, cfg); err != nil {
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
		runtime.LockOSThread()

		context := new(daemon.Context)
		if *pidfile != "" {
			context.PidFileName = *pidfile
			context.PidFilePerm = 0644
		}

		if runAsUser != nil {
			uid, err := strconv.ParseInt(runAsUser.Uid, 10, 0)
			if err != nil {
				log.Fatal(err)
			}

			gid, err := strconv.ParseInt(runAsUser.Gid, 10, 0)
			if err != nil {
				log.Fatal(err)
			}

			context.Credential = &syscall.Credential{
				Uid: uint32(uid),
				Gid: uint32(gid),
			}
		}

		child, _ := context.Reborn()

		if child != nil {
			return
		}
		defer context.Release()

		runtime.UnlockOSThread()
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
