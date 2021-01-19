package talker

import "context"

// Talker represents various ways to communicate
type Talker interface {
	IsConnected() bool
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	Send(ctx context.Context, source string, author string, channelID int, message string, optional string) error
	Subscribe(ctx context.Context, onMessage func(interface{}) error) error
}
