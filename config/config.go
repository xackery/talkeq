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

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) error {
	var err error
	d.Duration, err = time.ParseDuration(string(text))
	return err
}

// Config represents a configuration parse
type Config struct {
	Debug              bool
	IsKeepAliveEnabled bool     `toml:"keep_alive"`
	KeepAliveRetry     duration `toml:"keep_alive_retry"`
	Discord            Discord
	Telnet             Telnet
	EQLog              EQLog
	PEQEditor          PEQEditor `toml:"peq_editor"`
	Nats               Nats
	UsersDatabasePath  string `toml:"users_database"`
	GuildsDatabasePath string `toml:"guilds_database"`
}

// Discord represents config settings for discord
type Discord struct {
	IsEnabled       bool           `toml:"enabled"`
	OOC             DiscordChannel `toml:"ooc"`
	Auction         DiscordChannel `toml:"auction"`
	Guild           DiscordChannel `toml:"guild"`
	Shout           DiscordChannel `toml:"shout"`
	General         DiscordChannel `toml:"general"`
	Admin           DiscordChannel `toml:"admin"`
	PEQEditorSQLLog DiscordChannel `toml:"peq_editor_sql_log"`
	Token           string         `toml:"bot_token"`
	ServerID        string         `toml:"server_id"`
	ClientID        string         `toml:"client_id"`
	BotStatus       string         `toml:"bot_status"`
}

// DiscordChannel represents a discord channel
type DiscordChannel struct {
	SendChannelID   string `toml:"send_channel_id"`
	ListenChannelID string `toml:"listen_channel_id"`
}

// Telnet represents config settings for telnet
type Telnet struct {
	IsEnabled               bool `toml:"enabled"`
	IsLegacy                bool `toml:"legacy"`
	Host                    string
	Username                string
	Password                string
	ItemURL                 string   `toml:"item_url"`
	IsServerAnnounceEnabled bool     `toml:"announce_server_status"`
	MessageDeadline         duration `toml:"message_deadline"`
	IsOOCAuctionEnabled     bool     `toml:"convert_ooc_auction"`
}

// Nats represents config settings for NATS
type Nats struct {
	IsEnabled           bool `toml:"enabled"`
	Host                string
	IsOOCAuctionEnabled bool   `toml:"convert_ooc_auction"`
	ItemURL             string `toml:"item_url"`
}

// EQLog represents config settings for the EQ live eqlog file
type EQLog struct {
	IsEnabled                   bool   `toml:"enabled"`
	Path                        string `toml:"path"`
	IsGeneralChatAuctionEnabled bool   `toml:"convert_general_auction"`
	IsAuctionEnabled            bool   `toml:"listen_auction"`
	IsOOCEnabled                bool   `toml:"listen_ooc"`
	IsShoutEnabled              bool   `toml:"listen_shout"`
	IsGeneralEnabled            bool   `toml:"listen_general"`
}

// PEQEditor represents config settings for the PEQ editor service
type PEQEditor struct {
	SQL PEQEditorSQL `toml:"sql"`
}

// PEQEditorSQL is for config settings specific to the PEQ Editor SQL service
type PEQEditorSQL struct {
	IsEnabled   bool   `toml:"enabled"`
	Path        string `toml:"path"`
	FilePattern string `toml:"file_pattern"`
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

	if cfg.UsersDatabasePath == "" {
		cfg.UsersDatabasePath = "./users.txt"
	}

	if cfg.GuildsDatabasePath == "" {
		cfg.GuildsDatabasePath = "./guilds.txt"
	}

	return &cfg, nil
}
