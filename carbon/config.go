package carbon

import (
	"time"

	"github.com/lomik/go-carbon/config"
)

type commonConfig struct {
	User        string `toml:"user"`
	Logfile     string `toml:"logfile"`
	LogLevel    string `toml:"log-level"`
	GraphPrefix string `toml:"graph-prefix"`
	MaxCPU      int    `toml:"max-cpu"`
}

type whisperConfig struct {
	DataDir             string `toml:"data-dir"`
	Schemas             string `toml:"schemas-file"`
	Aggregation         string `toml:"aggregation-file"`
	Workers             int    `toml:"workers"`
	MaxUpdatesPerSecond int    `toml:"max-updates-per-second"`
	Enabled             bool   `toml:"enabled"`
}

type cacheConfig struct {
	MaxSize     int `toml:"max-size"`
	InputBuffer int `toml:"input-buffer"`
}

type udpConfig struct {
	Listen        string `toml:"listen"`
	Enabled       bool   `toml:"enabled"`
	LogIncomplete bool   `toml:"log-incomplete"`
}

type tcpConfig struct {
	Listen  string `toml:"listen"`
	Enabled bool   `toml:"enabled"`
}

type carbonlinkConfig struct {
	Listen       string           `toml:"listen"`
	Enabled      bool             `toml:"enabled"`
	ReadTimeout  *config.Duration `toml:"read-timeout"`
	QueryTimeout *config.Duration `toml:"query-timeout"`
}

type pprofConfig struct {
	Listen  string `toml:"listen"`
	Enabled bool   `toml:"enabled"`
}

// Config ...
type Config struct {
	Common     commonConfig     `toml:"common"`
	Whisper    whisperConfig    `toml:"whisper"`
	Cache      cacheConfig      `toml:"cache"`
	Udp        udpConfig        `toml:"udp"`
	Tcp        tcpConfig        `toml:"tcp"`
	Pickle     tcpConfig        `toml:"pickle"`
	Carbonlink carbonlinkConfig `toml:"carbonlink"`
	Pprof      pprofConfig      `toml:"pprof"`
}

// NewConfig creates and return new instance of carbon config
func NewConfig() *Config {
	cfg := &Config{
		Common: commonConfig{
			Logfile:     "/var/log/go-carbon/go-carbon.log",
			LogLevel:    "info",
			GraphPrefix: "carbon.agents.{host}.",
			MaxCPU:      1,
			User:        "",
		},
		Whisper: whisperConfig{
			DataDir:             "/data/graphite/whisper/",
			Schemas:             "/data/graphite/schemas",
			Aggregation:         "",
			MaxUpdatesPerSecond: 0,
			Enabled:             true,
			Workers:             1,
		},
		Cache: cacheConfig{
			MaxSize:     1000000,
			InputBuffer: 51200,
		},
		Udp: udpConfig{
			Listen:        ":2003",
			Enabled:       true,
			LogIncomplete: false,
		},
		Tcp: tcpConfig{
			Listen:  ":2003",
			Enabled: true,
		},
		Pickle: tcpConfig{
			Listen:  ":2004",
			Enabled: true,
		},
		Carbonlink: carbonlinkConfig{
			Listen:  "127.0.0.1:7002",
			Enabled: true,
			ReadTimeout: &config.Duration{
				Duration: 30 * time.Second,
			},
			QueryTimeout: &config.Duration{
				Duration: 100 * time.Millisecond,
			},
		},
		Pprof: pprofConfig{
			Listen:  "localhost:7007",
			Enabled: false,
		},
	}

	return cfg
}
