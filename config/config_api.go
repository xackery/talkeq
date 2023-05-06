package config

import (
	"fmt"

	"github.com/xackery/talkeq/tlog"
)

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

// Verify checks if config looks valid
func (c *API) Verify() error {
	if !c.IsEnabled {
		return nil
	}

	if c.APIRegister.IsEnabled {
		if len(c.APIRegister.RegistrationDatabasePath) == 0 {
			return fmt.Errorf("apiregister: registration path cannot be empty")
		}
	}

	if c.Host == "" {
		tlog.Debugf("[api] host was empty, defaulting to 127.0.0.1:9933")
		c.Host = "127.0.0.1:9933"
	}

	return nil
}
