package eqlog

import (
	"context"
	"fmt"

	"github.com/xackery/talkeq/model"
)

// SendMessage sends a message to discord
func (e *Endpoint) SendMessage(ctx context.Context, req *model.ChatMessage) (err error) {

	_, err = e.runQuery(ctx, "SendMessage", req)
	return
}

func (e *Endpoint) onSendMessage(ctx context.Context, req *model.ChatMessage) (resp *model.ChatMessage, err error) {
	err = fmt.Errorf("not supported")
	return
}
