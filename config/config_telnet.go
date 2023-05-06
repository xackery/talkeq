package config

import (
	"fmt"
	"text/template"
)

// Telnet represents config settings for telnet
type Telnet struct {
	IsEnabled               bool    `toml:"enabled" desc:"Enable Telnet"`
	IsLegacy                bool    `toml:"legacy" desc:"EQEMU servers that run 0.8.0 versions need this set to true for item link support, everyone running any newer versions can leave it default (false)"`
	Host                    string  `toml:"host" desc:"Address where telnet is found. By default, newer telnet clients will auto success on 127.0.0.1:9000"`
	Username                string  `toml:"username" desc:"Optional. Username to connect to telnet to. (By default, newer telnet clients will auto succeed if localhost)"`
	Password                string  `toml:"password" desc:"Optional. Password to connect to telnet to. (By default, newer telnet clients will auto succeed if localhost)"`
	Routes                  []Route `toml:"routes" desc:"Routes from telnet to other services"`
	ItemURL                 string  `toml:"item_url" desc:"Optional. Converts item URLs to provided field. defaults to allakhazam. To disable, change to \n# default: \"http://everquest.allakhazam.com/db/item.html?item=\""`
	IsServerAnnounceEnabled bool    `toml:"announce_server_status" desc:"Optional. Annunce when a server changes state to OOC channel (Server UP/Down)"`
	IsOOCAuctionEnabled     bool    `toml:"convert_ooc_auction" desc:"if a OOC message uses prefix WTS or WTB, convert them into auction"`
}

// TelnetEntry represents telnet event pattern detection
type TelnetEntry struct {
	ChannelID              string `toml:"channel_id" desc:"channel id to relay telnet event to"`
	Regex                  string `toml:"regex" desc:"regex to look for in message"`
	MessagePattern         string `toml:"pattern" desc:"Pattern to send message\n# Variables: {{.Msg}}, {{.Author}}, {{.ChannelNumber}}, {{.RegexGroup1}}, {{.RegexGroup2}} etc for submatch () patterns"`
	MessagePatternTemplate *template.Template
}

// Verify checks if config looks valid
func (c *Telnet) Verify() error {
	if !c.IsEnabled {
		return nil
	}
	for i := range c.Routes {
		if c.Routes[i].ChannelID == "" {
			return fmt.Errorf("route %d: invalid channel id", i)
		}
		err := c.Routes[i].LoadMessagePattern()
		if err != nil {
			return fmt.Errorf("route %d: %w", i, err)
		}
	}
	return nil
}
