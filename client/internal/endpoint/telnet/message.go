package telnet

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/xackery/talkeq/model"
)

// SendMessage sends a message to discord
func (e *Endpoint) SendMessage(ctx context.Context, req *model.ChatMessage) (err error) {
	_, err = e.runQuery(ctx, "SendMessage", req)
	return
}

func (e *Endpoint) onSendMessage(ctx context.Context, req *model.ChatMessage) (resp *model.ChatMessage, err error) {
	resp = &model.ChatMessage{}
	if !e.isConnected {
		err = fmt.Errorf("not connected")
		return
	}
	if req.ChannelNumber == 0 {
		req.ChannelNumber = 260
	}

	err = e.onSendln(fmt.Sprintf("emote world %d %s says from %s, '%s'", req.ChannelNumber, req.Author, req.SourceEndpoint, req.Message))
	if err != nil {
		err = errors.Wrap(err, "failed to send message")
		return
	}
	return
}
