package config

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sort"
	"text/template"
	"time"

	"github.com/jbsmith7741/toml"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

// Config represents a configuration parse
type Config struct {
	Debug              bool      `toml:"debug" desc:"TalkEQ Configuration\n\n# Debug messages are displayed. This will cause console to be more verbose, but also more informative"`
	IsKeepAliveEnabled bool      `toml:"keep_alive" desc:"Keep all connections alive?\n# If false, endpoint disconnects will not self repair\n# Not recommended to turn off except in advanced cases"`
	KeepAliveRetry     string    `toml:"keep_alive_retry" desc:"How long before retrying to connect (requires keep_alive = true)\n# default: 10s"`
	UsersDatabasePath  string    `toml:"users_database" desc:"Users by ID are mapped to their display names via the raw text file called users database\n# If users database file does not exist, a new one is created\n# This file is actively monitored. if you edit it while talkeq is running, it will reload the changes instantly\n# This file overrides the IGN: playerName role tags in discord\n# If a user is found on this list, it will fall back to check for IGN tags"`
	GuildsDatabasePath string    `toml:"guilds_database" desc:"** Only supported by NATS **\n# Guilds by ID are mapped to their database ID via the raw text file called guilds database\n# If guilds database file does not exist, and NATS is enabled, a new one is created\n# This file is actively monitored. if you edit it while talkeq is running, it will reload the changes instantly"`
	API                API       `toml:"api" desc:"API is a service to allow external tools to talk to TalkEQ via HTTP requests.\n# It uses Restful style (JSON) with a /api suffix for all endpoints"`
	Discord            Discord   `toml:"discord" desc:"Discord is a chat service that you can listen and relay EQ chat with"`
	Telnet             Telnet    `toml:"telnet" desc:"Telnet is a service eqemu/server can use, that relays messages over"`
	EQLog              EQLog     `toml:"eqlog" desc:"EQ Log is used to parse everquest client logs. Primarily for live EQ, non server owners"`
	PEQEditor          PEQEditor `toml:"peq_editor"`
	Nats               Nats      `toml:"nats" desc:"NATS is a custom alternative to telnet\n# that a very limited number of eqemu\n# servers utilize. Chances are, you can ignore"`
	SQLReport          SQLReport `toml:"sql_report" desc:"SQL Report can be used to show stats on discord\n# An ideal way to set this up is create a private voice channel\n# Then bind it to various queries"`
}

// API represents an API listening service
type API struct {
	IsEnabled   bool        `toml:"enabled" desc:"Enable API service"`
	Host        string      `toml:"host" desc:"What address and port to bind to (default is 127.0.0.1, so only local traffic can talk to it)"`
	APIRegister APIRegister `toml:"register" desc:"!register command"`
}

// APIRegister is used for Register command management
type APIRegister struct {
	IsEnabled                bool   `toml:"enabled" desc:"Enable !register command"`
	RegistrationDatabasePath string `toml:"registration_database" desc:"When a player requests to register, this database stores the request"`
}

// Discord represents config settings for discord
type Discord struct {
	IsEnabled       bool           `toml:"enabled" desc:"Enable Discord"`
	Token           string         `toml:"bot_token" desc:"Required. Found at https://discordapp.com/developers/ under your app's bot's section"`
	ServerID        string         `toml:"server_id" desc:"Required. In Discord, right click the circle button representing your server, and Copy ID, and paste it here."`
	ClientID        string         `toml:"client_id" desc:"Required. Found at https://discordapp.com/developers/ under your app's main page"`
	BotStatus       string         `toml:"bot_status" desc:"Status to show below bot. e.g. \"Playing EQ: 123 Online\"\n# {{.PlayerCount}} to show playercount"`
	CommandChannels []string       `toml:"command_channels" desc:"Commands are parsed in provided channel ids"`
	Routes          []DiscordRoute `toml:"routes" desc:"When a message is created in discord, how to route it"`
}

// DiscordRoute is custom for discord triggering
type DiscordRoute struct {
	IsEnabled              bool           `toml:"enabled" desc:"Is route enabled?"`
	Trigger                DiscordTrigger `toml:"discord_trigger" desc:"condition to trigger route"`
	Target                 string         `toml:"target" desc:"target service, e.g. telnet"`
	ChannelID              string         `toml:"channel_id" desc:"Destination channel ID, e.g. OOC is 260"`
	GuildID                string         `toml:"guild_id" desc:"Optional, Destination guild ID"`
	MessagePattern         string         `toml:"message_pattern" desc:"Destination message in. E.g. {{.Name}} says {{.ChannelName}}, '{{.Message}}"`
	messagePatternTemplate *template.Template
}

// DiscordTrigger is custom discord triggering
type DiscordTrigger struct {
	ChannelID string `toml:"channel_id" desc:"source channel ID to trigger event"`
}

// MessagePatternTemplate returns a template for provided route
func (r *DiscordRoute) MessagePatternTemplate() *template.Template {
	return r.messagePatternTemplate
}

// Route is how to route telnet messages
type Route struct {
	IsEnabled              bool    `toml:"enabled" desc:"Is route enabled?"`
	Trigger                Trigger `toml:"trigger" desc:"condition to trigger route"`
	Target                 string  `toml:"target" desc:"target service, e.g. telnet"`
	ChannelID              string  `toml:"channel_id" desc:"Destination channel ID, e.g. OOC is 260"`
	GuildID                string  `toml:"guild_id" desc:"Optional, Destination guild ID"`
	MessagePattern         string  `toml:"message_pattern" desc:"Destination message in. E.g. {{.Name}} says {{.ChannelName}}, '{{.Message}}"`
	messagePatternTemplate *template.Template
}

// Trigger is a regex pattern matching
type Trigger struct {
	Regex        string `toml:"telnet_pattern" desc:"Input telnet trigger regex"`
	NameIndex    int    `toml:"name_index" desc:"Name is found in this regex index grouping"`
	MessageIndex int    `toml:"message_index" desc:"Message is found in this regex index grouping"`
	Custom       string `toml:"custom,omitempty" dec:"Custom event defined in code"`
}

// MessagePatternTemplate returns a template for provided route
func (r *Route) MessagePatternTemplate() *template.Template {
	return r.messagePatternTemplate
}

// Telnet represents config settings for telnet
type Telnet struct {
	IsEnabled               bool    `toml:"enabled" desc:"Enable Telnet"`
	IsLegacy                bool    `toml:"legacy" desc:"EQEMU servers that run 0.8.0 versions need this for item link support"`
	Host                    string  `toml:"host" desc:"Host where telnet is found"`
	Username                string  `toml:"username" desc:"Optional. Username to connect to telnet to. (By default, newer telnet clients will auto succeed if localhost)"`
	Password                string  `toml:"password" desc:"Optional. Password to connect to telnet to. (By default, newer telnet clients will auto succeed if localhost)"`
	Routes                  []Route `toml:"routes" desc:"Routes from telnet to other services"`
	ItemURL                 string  `toml:"item_url" desc:"Optional. Converts item URLs to provided field. defaults to allakhazam. To disable, change to \n# default: \"http://everquest.allakhazam.com/db/item.html?item=\""`
	IsServerAnnounceEnabled bool    `toml:"announce_server_status" desc:"Optional. Annunce when a server changes state to OOC channel (Server UP/Down)"`
	MessageDeadline         string  `toml:"message_deadline" desc:"How long to wait for messages. (Advanced users only)\n# defaut: 10s"`
	messageDeadlineDuration time.Duration
	IsOOCAuctionEnabled     bool           `toml:"convert_ooc_auction" desc:"if a OOC message uses prefix WTS or WTB, convert them into auction"`
	Entries                 []*TelnetEntry `toml:"entries" desc:"Entries is full of custom pattern detection. Useful for emotes and custom messages"`
}

// TelnetEntry represents telnet event pattern detection
type TelnetEntry struct {
	ChannelID              string `toml:"channel_id" desc:"channel id to relay telnet event to"`
	Regex                  string `toml:"regex" desc:"regex to look for in message"`
	MessagePattern         string `toml:"pattern" desc:"Pattern to send message\n# Variables: {{.Msg}}, {{.Author}}, {{.ChannelNumber}}, {{.RegexGroup1}}, {{.RegexGroup2}} etc for submatch () patterns"`
	MessagePatternTemplate *template.Template
}

// Nats represents config settings for NATS
type Nats struct {
	IsEnabled           bool `toml:"enabled"`
	Host                string
	IsOOCAuctionEnabled bool    `toml:"convert_ooc_auction"`
	ItemURL             string  `toml:"item_url"`
	Routes              []Route `toml:"routes" desc:"Routes from nats to other services"`
}

// EQLog represents config settings for the EQ live eqlog file
type EQLog struct {
	IsEnabled                   bool    `toml:"enabled"`
	Path                        string  `toml:"path"`
	Routes                      []Route `toml:"routes" desc:"Routes from EQLog to other services"`
	IsGeneralChatAuctionEnabled bool    `toml:"convert_general_auction" desc:"convert WTS and WTB messages in general chat to auction channel"`
}

// PEQEditor represents config settings for the PEQ editor service
type PEQEditor struct {
	SQL    PEQEditorSQL `toml:"sql"`
	Routes []Route      `toml:"routes" desc:"Routes from PEQ Editor to other services"`
}

// PEQEditorSQL is for config settings specific to the PEQ Editor SQL service
type PEQEditorSQL struct {
	IsEnabled   bool    `toml:"enabled"`
	Path        string  `toml:"path"`
	FilePattern string  `toml:"file_pattern"`
	Routes      []Route `toml:"routes" desc:"Routes from peq editor to other services"`
}

// SQLReport is used for reporting SQL data to discord
type SQLReport struct {
	IsEnabled bool `toml:"enabled"`
	Host      string
	Username  string
	Password  string
	Database  string
	Entries   []*SQLReportEntries `toml:"entries"`
	Routes    []SQLReportRoute    `toml:"routes" desc:"Routes from telnet to other services"`
}

// SQLReportRoute is how to route SQL report messages
type SQLReportRoute struct {
	IsEnabled              bool             `toml:"enabled" desc:"Is route enabled?"`
	Trigger                SQLReportTrigger `toml:"trigger" desc:"condition to trigger route"`
	Target                 string           `toml:"target" desc:"target service, e.g. telnet"`
	ChannelID              string           `toml:"channel_id" desc:"Destination channel ID, e.g. OOC is 260"`
	GuildID                string           `toml:"guild_id" desc:"Optional, Destination guild ID"`
	MessagePattern         string           `toml:"message_pattern" desc:"Destination message in. E.g. {{.Name}} says {{.ChannelName}}, '{{.Message}}"`
	messagePatternTemplate *template.Template
}

// SQLReportTrigger is a regex pattern matching
type SQLReportTrigger struct {
	Query string `toml:"query" desc:"query to send to SQL"`
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

// KeepAliveRetryDuration returns the converted retry rate
func (c *Config) KeepAliveRetryDuration() time.Duration {
	retryDuration, err := time.ParseDuration(c.KeepAliveRetry)
	if err != nil {
		return 10 * time.Second
	}

	if retryDuration < 10*time.Second {
		return 10 * time.Second
	}
	return retryDuration
}

// MessageDeadlineDuration returns the converted retry rate
func (c *Telnet) MessageDeadlineDuration() time.Duration {
	deadlineDuration, err := time.ParseDuration(c.MessageDeadline)
	if err != nil {
		return 10 * time.Second
	}

	if deadlineDuration < 10*time.Second {
		return 10 * time.Second
	}
	return deadlineDuration
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

		enc := toml.NewEncoder(f)
		enc.Encode(getDefaultConfig())

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

	fw, err := os.Create("talkeq2.toml")
	if err != nil {
		return nil, fmt.Errorf("talkeq: %w", err)
	}
	defer fw.Close()

	enc := toml.NewEncoder(fw)
	err = enc.Encode(cfg)
	if err != nil {
		return nil, fmt.Errorf("encode: %w", err)
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

func getDefaultConfig() Config {
	cfg := Config{
		Debug:              true,
		IsKeepAliveEnabled: true,
		KeepAliveRetry:     "10s",
		UsersDatabasePath:  "talkeq_users.toml",
		GuildsDatabasePath: "talkeq_guilds.txt",
	}
	cfg.API.IsEnabled = true
	cfg.API.Host = ":9933"
	cfg.API.APIRegister.IsEnabled = true
	cfg.API.APIRegister.RegistrationDatabasePath = "talkeq_register.toml"

	cfg.Discord.IsEnabled = true
	cfg.Discord.BotStatus = "EQ: {{.PlayerCount}} Online"
	cfg.Discord.Routes = append(cfg.Discord.Routes, DiscordRoute{
		IsEnabled: true,
		Trigger: DiscordTrigger{
			ChannelID: "INSERTOOCCHANNELHERE",
		},
		Target:         "telnet",
		ChannelID:      "260",
		MessagePattern: "emote world {{.ChannelID}} {{.Name}} says from discord, '{{.Message}}'",
	})

	cfg.Discord.Routes = append(cfg.Discord.Routes, DiscordRoute{
		IsEnabled: true,
		Trigger: DiscordTrigger{
			ChannelID: "INSERTOOCCHANNELHERE",
		},
		Target:         "nats",
		ChannelID:      "260",
		MessagePattern: "{{.Name}} says from discord, '{{.Message}}'",
	})

	cfg.Telnet.IsEnabled = true
	cfg.Telnet.Host = "127.0.0.1:9000"
	cfg.Telnet.ItemURL = "http://everquest.allakhazam.com/db/item.html?item="
	cfg.Telnet.IsServerAnnounceEnabled = true
	cfg.Telnet.MessageDeadline = "10s"
	cfg.Telnet.IsOOCAuctionEnabled = true
	cfg.Telnet.Routes = append(cfg.Telnet.Routes, Route{
		IsEnabled: true,
		Trigger: Trigger{
			Regex:        `(\w+) says ooc, '(.*)'`,
			NameIndex:    1,
			MessageIndex: 2,
		},
		Target:         "discord",
		ChannelID:      "260",
		MessagePattern: "{{.Name}} **OOC**: {{.Message}}",
	})

	cfg.Telnet.Routes = append(cfg.Telnet.Routes, Route{
		IsEnabled: true,
		Trigger: Trigger{
			Regex:        `(\w+) auctions, '(.*)'`,
			NameIndex:    1,
			MessageIndex: 2,
		},
		Target:         "discord",
		ChannelID:      "INSERTAUCTIONCHANNELHERE",
		MessagePattern: "{{.Name}} **auction**: {{.Message}}",
	})

	cfg.Telnet.Routes = append(cfg.Telnet.Routes, Route{
		IsEnabled: true,
		Trigger: Trigger{
			Regex:        `(\w+) general, '(.*)'`,
			NameIndex:    1,
			MessageIndex: 2,
		},
		Target:         "discord",
		ChannelID:      "INSERTGENERALCHANNELHERE",
		MessagePattern: "{{.Name}} **general**: {{.Message}}",
	})

	cfg.Telnet.Routes = append(cfg.Telnet.Routes, Route{
		IsEnabled: true,
		Trigger: Trigger{
			Regex:        `(\w+) BROADCASTS, '(.*)'`,
			NameIndex:    1,
			MessageIndex: 2,
		},
		Target:         "discord",
		ChannelID:      "INSERTOOCCHANNELHERE",
		MessagePattern: "{{.Name}} **BROADCAST**: {{.Message}}",
	})

	cfg.Telnet.Routes = append(cfg.Telnet.Routes, Route{
		IsEnabled: true,
		Trigger: Trigger{
			Custom: "serverup",
		},
		Target:         "discord",
		ChannelID:      "INSERTOOCCHANNELHERE",
		MessagePattern: "**Admin ooc:** Server is now UP",
	})
	cfg.Telnet.Routes = append(cfg.Telnet.Routes, Route{
		IsEnabled: true,
		Trigger: Trigger{
			Custom: "serverdown",
		},
		Target:         "discord",
		ChannelID:      "INSERTOOCCHANNELHERE",
		MessagePattern: "**Admin ooc:** Server is now DOWN",
	})

	cfg.EQLog.Path = `c:\Program Files\Everquest\Logs\eqlog_CharacterName_Server.txt`
	cfg.EQLog.Routes = append(cfg.EQLog.Routes, Route{
		IsEnabled: true,
		Trigger: Trigger{
			Regex:        `(\w+) says out of character, '(.*)'`,
			NameIndex:    1,
			MessageIndex: 2,
		},
		Target:         "discord",
		ChannelID:      "INSERTOOCCHANNELHERE",
		MessagePattern: "{{.Name}} **OOC**: {{.Message}}",
	})
	cfg.EQLog.Routes = append(cfg.EQLog.Routes, Route{
		IsEnabled: true,
		Trigger: Trigger{
			Regex:        `(\w+) auctions, '(.*)'`,
			NameIndex:    1,
			MessageIndex: 2,
		},
		Target:         "discord",
		ChannelID:      "INSERTAUCTIONCHANNELHERE",
		MessagePattern: "{{.Name}} **OOC**: {{.Message}}",
	})
	cfg.EQLog.Routes = append(cfg.EQLog.Routes, Route{
		IsEnabled: true,
		Trigger: Trigger{
			Regex:        `(\w+) says to general, '(.*)'`,
			NameIndex:    1,
			MessageIndex: 2,
		},
		Target:         "discord",
		ChannelID:      "INSERTGENERALCHANNELHERE",
		MessagePattern: "{{.Name}} **OOC**: {{.Message}}",
	})
	cfg.EQLog.Routes = append(cfg.EQLog.Routes, Route{
		IsEnabled: true,
		Trigger: Trigger{
			Regex:        `(\w+) shouts, '(.*)'`,
			NameIndex:    1,
			MessageIndex: 2,
		},
		Target:         "discord",
		ChannelID:      "INSERTSHOUTCHANNELHERE",
		MessagePattern: "{{.Name}} **OOC**: {{.Message}}",
	})
	cfg.EQLog.Routes = append(cfg.EQLog.Routes, Route{
		IsEnabled: true,
		Trigger: Trigger{
			Regex:        `(\w+) says to guild, '(.*)'`,
			NameIndex:    1,
			MessageIndex: 2,
		},
		Target:         "discord",
		ChannelID:      "INSERTGUILDCHANNELHERE",
		MessagePattern: "{{.Name}} **OOC**: {{.Message}}",
	})

	cfg.PEQEditor.Routes = append(cfg.EQLog.Routes, Route{
		IsEnabled: true,
		Trigger: Trigger{
			Regex:        `(.*)`,
			NameIndex:    0,
			MessageIndex: 1,
		},
		Target:         "discord",
		ChannelID:      "INSERPEQEDITORLOGCHANNELHERE",
		MessagePattern: "{{.Name}} **OOC**: {{.Message}}",
	})

	cfg.Nats.Host = "127.0.0.1:4222"
	cfg.Nats.IsOOCAuctionEnabled = true
	cfg.Nats.Routes = append(cfg.EQLog.Routes, Route{
		IsEnabled: true,
		Trigger: Trigger{
			Custom: "admin",
		},
		Target:         "discord",
		ChannelID:      "INSERTADMINCHANNELHERE",
		MessagePattern: "{{.Name}} **ADMIN**: {{.Message}}",
	})
	cfg.Nats.Routes = append(cfg.EQLog.Routes, Route{
		IsEnabled: true,
		Trigger: Trigger{
			Custom: "260",
		},
		Target:         "discord",
		ChannelID:      "INSERTOOCCHANNELHERE",
		MessagePattern: "{{.Name}} **OOC**: {{.Message}}",
	})

	cfg.PEQEditor.SQL.Path = "/var/www/peq/peqphpeditor/logs"
	cfg.PEQEditor.SQL.FilePattern = "sql_log_{{.Month}}-{{.Year}}.sql"

	cfg.SQLReport.Host = "127.0.0.1:3306"
	cfg.SQLReport.Username = "eqemu"
	cfg.SQLReport.Password = "eqemu"
	cfg.SQLReport.Database = "eqemu"
	return cfg
}
