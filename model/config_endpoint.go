package model

// NewConfigEndpoint creates a new configuration with default settings
func NewConfigEndpoint() (config *ConfigEndpoint) {
	config = &ConfigEndpoint{
		Discord: &ConfigEndpointDiscord{},
		Telnet:  &ConfigEndpointTelnet{},
		EQLog:   &ConfigEndpointEQLog{},
		NATS:    &ConfigEndpointNATS{},
		Routes:  make(map[string][]string),
	}
	return
}

// ConfigEndpoint is used to configure various endpoints
type ConfigEndpoint struct {
	Discord *ConfigEndpointDiscord
	Telnet  *ConfigEndpointTelnet
	EQLog   *ConfigEndpointEQLog
	NATS    *ConfigEndpointNATS
	Routes  map[string][]string
	ItemURL string
}

// ConfigEndpointDiscord configures discord
type ConfigEndpointDiscord struct {
	Enabled  bool
	Token    string
	ServerID string
}

// ConfigEndpointTelnet configures telnet
type ConfigEndpointTelnet struct {
	Enabled bool
	//Username based on telnet settings. Optional, if localhost
	Username string
	//Username based on telnet settings. Optional, if localhost
	Password string
	IP       string
	Port     string
}

// ConfigEndpointEQLog configures EQLog parsing
type ConfigEndpointEQLog struct {
	Enabled bool
	Path    string
}

// ConfigEndpointNATS configures NATS
type ConfigEndpointNATS struct {
	Enabled bool
	IP      string
	Port    string
}
