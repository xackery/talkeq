package request

import (
	"context"
)

// DiscordSend Request
type DiscordSend struct {
	Ctx       context.Context
	ChannelID string
	Message   string
}

// DiscordEdit Request
type DiscordEdit struct {
	Ctx       context.Context
	ChannelID string
	Message   string
}

// APICommand Request
type APICommand struct {
	Ctx                  context.Context
	FromDiscordName      string
	FromDiscordChannelID string
	FromDiscordNameID    string
	FromDiscordIGN       string
	Message              string
}

// EQLog originated from EQLog
type EQLog struct {
	Ctx                context.Context
	Action             string
	Target             int
	FromName           string
	Message            string
	ToDiscordChannelID string
	ToName             string
}

// TelnetSend request
type TelnetSend struct {
	Ctx     context.Context
	Message string
}

// PEQEditorSQL originated from PEQ Editor
type PEQEditorSQL struct {
	Ctx            context.Context
	Action         string
	Target         int
	FromName       string
	Message        string
	ChannelKeyword string
	ToName         string
}
