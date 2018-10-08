package endpoint

import (
	"context"

	"github.com/xackery/talkeq/model"
)

// Endpointer represents an input or output talkEQ talks to
type Endpointer interface {
	Connect(ctx context.Context) (err error)
	ConfigRead(ctx context.Context) (config *model.ConfigEndpoint, err error)
	ConfigUpdate(ctx context.Context, config *model.ConfigEndpoint) (err error)
	Close(ctx context.Context)
	SendMessage(ctx context.Context, message *model.ChatMessage) (err error)
}
