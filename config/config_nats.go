package config

import "fmt"

// Nats represents config settings for NATS
type Nats struct {
	IsEnabled           bool `toml:"enabled"`
	Host                string
	IsOOCAuctionEnabled bool    `toml:"convert_ooc_auction"`
	ItemURL             string  `toml:"item_url"`
	Routes              []Route `toml:"routes" desc:"Routes from nats to other services"`
}

// Verify checks if config looks valid
func (c *Nats) Verify() error {
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
