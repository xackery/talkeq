package config

import (
	"context"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// Config represents a configuration parse
type Config struct {
	Debug              bool
	IsKeepAliveEnabled bool `toml:"keep_alive"`
	Discord            Discord
	Telnet             Telnet
}

// Discord represents config settings for discord
type Discord struct {
	IsEnabled          bool   `toml:"enabled"`
	OOCListenChannelID string `toml:"ooc_listen_channel_id"`
	OOCSendChannelID   string `toml:"ooc_send_channel_id"`
	Token              string `toml:"bot_token"`
	ServerID           string `toml:"server_id"`
	ClientID           string `toml:"client_id"`
}

// Telnet represents config settings for telnet
type Telnet struct {
	IsEnabled bool
	Host      string
	Username  string
	Password  string
	ItemURL   string `toml:"item_url"`
}

// NewConfig creates a new configuration
func NewConfig(ctx context.Context) (*Config, error) {
	var f *os.File
	cfg := Config{}
	path := "talkeq.conf"

	isNewConfig := false
	fi, err := os.Stat(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, errors.Wrap(err, "config info")
		}
		f, err = os.Create(path)
		if err != nil {
			return nil, errors.Wrap(err, "create talkeq.conf")
		}
		fi, err = os.Stat(path)
		if err != nil {
			return nil, errors.Wrap(err, "new config info")
		}
		isNewConfig = true
	}
	if !isNewConfig {
		f, err = os.Open(path)
		if err != nil {
			return nil, errors.Wrap(err, "open config")
		}
	}

	defer f.Close()
	if fi.IsDir() {
		return nil, fmt.Errorf("talkeq.conf is a directory, should be a file")
	}

	if isNewConfig {
		_, err = f.WriteString(defaultConfig)
		if err != nil {
			return nil, errors.Wrap(err, "write new talkeq.conf")
		}
		fmt.Println("a new talkeq.conf file was created. Please open this file and configure talkeq, then run it again.")
		os.Exit(0)
	}

	_, err = toml.DecodeReader(f, &cfg)
	if err != nil {
		return nil, errors.Wrap(err, "decode talkeq.conf")
	}

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if cfg.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	return &cfg, nil
}
