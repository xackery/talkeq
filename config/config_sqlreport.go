package config

import (
	"fmt"
	"text/template"
	"time"
)

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

// Verify returns any errors while verifying config
func (c *SQLReport) Verify() error {
	var err error
	if !c.IsEnabled {
		return nil
	}
	for i, e := range c.Entries {
		e.Index = i

		e.RefreshDuration, err = time.ParseDuration(e.Refresh)
		if err != nil {
			return fmt.Errorf("refresh_duration is invalid %s for pattern %s: %w", e.Refresh, e.Pattern, err)
		}
		if e.RefreshDuration < 30*time.Second {
			return fmt.Errorf("duration %s is lower than 30s for sqlreport pattern %s", e.Refresh, e.Pattern)
		}

		e.PatternTemplate, err = template.New("pattern").Parse(e.Pattern)
		if err != nil {
			return fmt.Errorf("parse sqlreport pattern %s: %w", e.Pattern, err)
		}
		e.NextReport = time.Now()
	}
	return nil
}
