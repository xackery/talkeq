package config

import (
	"fmt"
	"text/template"
)

// Route is how to route telnet messages
type Route struct {
	IsEnabled              bool    `toml:"enabled" desc:"Is route enabled?"`
	Trigger                Trigger `toml:"trigger" desc:"condition to trigger route"`
	Target                 string  `toml:"target" desc:"target service, e.g. telnet"`
	ChannelID              string  `toml:"channel_id" desc:"Destination channel ID"`
	GuildID                string  `toml:"guild_id,omitempty" desc:"Optional, Destination guild ID"`
	MessagePattern         string  `toml:"message_pattern" desc:"Destination message in. E.g. {{.Name}} says {{.ChannelName}}, '{{.Message}}"`
	messagePatternTemplate *template.Template
}

// MessagePatternTemplate returns a template for provided route
func (r *Route) MessagePatternTemplate() *template.Template {
	return r.messagePatternTemplate
}

// LoadMessagePattern is called after config is loaded, and verified patterns are valid
func (r *Route) LoadMessagePattern() error {
	if !r.IsEnabled {
		return nil
	}
	var err error
	r.messagePatternTemplate, err = template.New("root").Parse(r.MessagePattern)
	if err != nil {
		return fmt.Errorf("failed to parse: %w", err)
	}
	return nil
}
