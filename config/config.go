package config

import (
	"bytes"
	"flag"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

var configFile string
var printDefaultConfig bool

func init() {
	flag.StringVar(&configFile, "config", "", "Filename of config")
	flag.BoolVar(&printDefaultConfig, "config-print-default", false, "Print default config")
}

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

// Parse ...
func Parse(cfg interface{}) error {
	if printDefaultConfig {
		if err := PrintConfig(cfg); err != nil {
			return err
		}
		os.Exit(0)
	}

	return ParseConfig(configFile, cfg)
}
