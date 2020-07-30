package config

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sort"
	"text/template"
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
	SQLReport          SQLReport `toml:"sql_report"`
	UsersDatabasePath  string    `toml:"users_database"`
	GuildsDatabasePath string    `toml:"guilds_database"`
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
	Broadcast       DiscordChannel `toml:"broadcast"`
	Emote           DiscordChannel `toml:"emote"`
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
	ItemURL                 string         `toml:"item_url"`
	IsServerAnnounceEnabled bool           `toml:"announce_server_status"`
	MessageDeadline         duration       `toml:"message_deadline"`
	IsOOCAuctionEnabled     bool           `toml:"convert_ooc_auction"`
	Entries                 []*TelnetEntry `toml:"entries"`
}

// TelnetEntry represents telnet event pattern detection
type TelnetEntry struct {
	ChannelID              string `toml:"channel_id"`
	Regex                  string
	MessagePattern         string `toml:"pattern"`
	MessagePatternTemplate *template.Template
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

// SQLReport is used for reporting SQL data to discord
type SQLReport struct {
	IsEnabled bool `toml:"enabled"`
	Host      string
	Username  string
	Password  string
	Database  string
	Entries   []*SQLReportEntries `toml:"entries"`
}

//SQLReportEntries is used for entries in a sql report
type SQLReportEntries struct {
	ChannelID       string `toml:"channel_id"`
	Query           string
	Pattern         string
	PatternTemplate *template.Template
	Refresh         string
	RefreshDuration time.Duration
	// Last time a report was successfully sent
	NextReport time.Time
	Text       string
	Index      int
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

	if cfg.SQLReport.IsEnabled {
		for i, e := range cfg.SQLReport.Entries {
			e.Index = i

			e.RefreshDuration, err = time.ParseDuration(e.Refresh)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to parse duration %s for sqlreport pattern %s", e.Refresh, e.Pattern)
			}
			if e.RefreshDuration < 30*time.Second {
				return nil, fmt.Errorf("duration %s is lower than 30s for sqlreport pattern %s", e.Refresh, e.Pattern)
			}

			if e.PatternTemplate, err = template.New("pattern").Parse(e.Pattern); err != nil {
				return nil, errors.Wrapf(err, "failed to parse pattern %s for sqlreport", e.Pattern)
			}
			e.NextReport = time.Now()
		}
	}
	if cfg.Telnet.IsEnabled {
		for _, e := range cfg.Telnet.Entries {
			if e.MessagePatternTemplate, err = template.New("pattern").Parse(e.MessagePattern); err != nil {
				return nil, errors.Wrapf(err, "failed to parse pattern %s for telnet", e.MessagePattern)
			}
		}
	}

	sort.SliceStable(cfg.SQLReport.Entries, func(i, j int) bool {
		return cfg.SQLReport.Entries[i].Index > cfg.SQLReport.Entries[j].Index
	})

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
