package config

import (
	"fmt"
	"text/template"
)

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
	ChannelID              string         `toml:"channel_id" desc:"Destination discord channel ID, right click a channel in discord and Copy ID to paste here"`
	GuildID                string         `toml:"guild_id" desc:"Optional, Destination guild ID"`
	MessagePattern         string         `toml:"message_pattern" desc:"Destination message in. E.g. {{.Name}} says {{.ChannelName}}, '{{.Message}}"`
	messagePatternTemplate *template.Template
}

// DiscordTrigger is custom discord triggering
type DiscordTrigger struct {
	ChannelID string `toml:"channel_id" desc:"source channel ID to trigger event"`
}

// Verify checks if config looks valid
func (c *Discord) Verify() error {
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

// MessagePatternTemplate returns a template for provided route
func (r *DiscordRoute) MessagePatternTemplate() *template.Template {
	return r.messagePatternTemplate
}

// LoadMessagePattern is called after config is loaded, and verified patterns are valid
func (r *DiscordRoute) LoadMessagePattern() error {
	var err error
	r.messagePatternTemplate, err = template.New("root").Parse(r.MessagePattern)
	if err != nil {
		return fmt.Errorf("failed to parse: %w", err)
	}
	return nil
}
