package config

import "fmt"

// EQLog represents config settings for the EQ live eqlog file
type EQLog struct {
	IsEnabled                   bool    `toml:"enabled"`
	Path                        string  `toml:"path"`
	Routes                      []Route `toml:"routes" desc:"Routes from EQLog to other services"`
	IsGeneralChatAuctionEnabled bool    `toml:"convert_general_auction" desc:"convert WTS and WTB messages in general chat to auction channel"`
}

// Verify checks if config looks valid
func (c *EQLog) Verify() error {
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
