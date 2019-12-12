package config

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// Config represents a configuration parse
type Config struct {
	Debug              bool
	IsKeepAliveEnabled bool          `toml:"keep_alive"`
	KeepAliveRetry     time.Duration `toml:"keep_alive_retry"`
	Discord            Discord
	Telnet             Telnet
	EQLog              EQLog
}

// Discord represents config settings for discord
type Discord struct {
	IsEnabled bool           `toml:"enabled"`
	OOC       DiscordChannel `toml:"ooc"`
	Auction   DiscordChannel `toml:"auction"`
	Guild     DiscordChannel `toml:"guild"`
	Shout     DiscordChannel `toml:"shout"`
	General   DiscordChannel `toml:"general"`
	Token     string         `toml:"bot_token"`
	ServerID  string         `toml:"server_id"`
	ClientID  string         `toml:"client_id"`
	BotStatus string         `toml:"bot_status"`
}

// DiscordChannel represents a discord channel
type DiscordChannel struct {
	SendChannelID   string `toml:"send_channel_id"`
	ListenChannelID string `toml:"listen_channel_id"`
}

// Telnet represents config settings for telnet
type Telnet struct {
	IsEnabled               bool `toml:"enabled"`
	Host                    string
	Username                string
	Password                string
	ItemURL                 string        `toml:"item_url"`
	IsServerAnnounceEnabled bool          `toml:"announce_server_status"`
	MessageDeadline         time.Duration `toml:"message_deadline"`
}

// EQLog represents config settings for the EQ live eqlog file
type EQLog struct {
	IsEnabled                   bool `toml:"enabled"`
	Path                        string
	IsGeneralChatAuctionEnabled bool `toml:"convert_general_auction"`
	IsAuctionEnabled            bool `toml:"listen_auction"`
	IsOOCEnabled                bool `toml:"listen_ooc"`
	IsShoutEnabled              bool `toml:"listen_shout"`
	IsGeneralEnabled            bool `toml:"listen_general"`
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
		if runtime.GOOS == "windows" {
			option := ""
			fmt.Println("press a key then enter to exit.")
			fmt.Scan(&option)
		}
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
