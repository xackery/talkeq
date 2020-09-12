package config

import "fmt"

// PEQEditor represents config settings for the PEQ editor service
type PEQEditor struct {
	IsEnabled bool         `toml:"enabled"`
	SQL       PEQEditorSQL `toml:"sql"`
}

// PEQEditorSQL is for config settings specific to the PEQ Editor SQL service
type PEQEditorSQL struct {
	IsEnabled   bool    `toml:"enabled"`
	Path        string  `toml:"path"`
	FilePattern string  `toml:"file_pattern"`
	Routes      []Route `toml:"routes" desc:"Routes from peq editor to other services"`
}

// Verify checks if config looks valid
func (c *PEQEditor) Verify() error {
	if !c.IsEnabled {
		return nil
	}
	if c.SQL.IsEnabled {
		if len(c.SQL.Path) == 0 {
			return fmt.Errorf("sql: path is empty")
		}
		if len(c.SQL.FilePattern) == 0 {
			return fmt.Errorf("sql: file pattern is empty")
		}
		for i := range c.SQL.Routes {
			if c.SQL.Routes[i].ChannelID == "" {
				return fmt.Errorf("route %d: invalid channel id", i)
			}
			err := c.SQL.Routes[i].LoadMessagePattern()
			if err != nil {
				return fmt.Errorf("route %d: %w", i, err)
			}
		}
	}
	return nil
}
