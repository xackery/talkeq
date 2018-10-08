package nats

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/nats-io/go-nats"
	"github.com/pkg/errors"
	"github.com/xackery/talkeq/model"
	"github.com/xackery/talkeq/pb"
)

// SendMessage sends a message to discord
func (e *Endpoint) SendMessage(ctx context.Context, req *model.ChatMessage) (err error) {
	return
}

func (e *Endpoint) onSendMessage(ctx context.Context, req *model.ChatMessage) (resp *model.ChatMessage, err error) {
	if !e.isConnected {
		err = fmt.Errorf("not connected")
		return
	}
	message := &pb.ChannelMessage{
		ChanNum: int32(req.ChannelNumber),
		From:    req.Author,
	}
	req.Message = fmt.Sprintf("%s says from %s, '%s'", req.Author, req.SourceEndpoint, req.Message)
	msg, err := proto.Marshal(message)
	if err != nil {
		err = errors.Wrap(err, "failed to marshal message")
		return
	}

	err = e.conn.Publish("world.channel_message.in", msg)
	if err != nil {
		err = errors.Wrap(err, "failed to publish message")
		return
	}
	return
}

// ChannelMessageRead consumes messages from discord
func (e *Endpoint) ChannelMessageRead(req *nats.Msg) {
	ctx := context.Background()
	var err error
	_, err = e.runQuery(ctx, "ChannelMessageRead", req)
	if err != nil {
		logger := model.NewLogger(ctx)
		logger.Error().Err(err).Msg("error while reading discord message")
		return
	}
	return
}

func (e *Endpoint) onChannelMessageRead(ctx context.Context, msg *nats.Msg) (err error) {

	message := &pb.ChannelMessage{}
	err = proto.Unmarshal(msg.Data, message)
	if err != nil {
		err = errors.Wrap(err, "failed to unmarshal channel message")
		return
	}
	req := &model.ChatMessage{
		Author:         message.From,
		ChannelNumber:  int(message.ChanNum),
		SourceEndpoint: "nats",
		Message:        message.Message,
	}

	req.Author = strings.Replace(req.Author, "_", " ", -1)
	if strings.Contains(req.Author, " ") {
		req.Author = fmt.Sprintf("%s [%s]", req.Author[:strings.Index(req.Author, " ")], req.Author[strings.Index(req.Author, " ")+1:])
	}
	req.Author = model.Alphanumeric(req.Author) //purify name to be alphanumeric

	if strings.Contains(req.Message, "Summoning you to") { //GM messages are relaying to discord!
		return
	}
	req.Message = e.onConvertLinks(req.Message)

	err = e.manager.ChatMessage(ctx, req)
	if err != nil {
		err = errors.Wrap(err, "manager erorr")
		return
	}
	return
}

func (e *Endpoint) sanitize(data string) (sData string) {
	sData = data
	sData = strings.Replace(sData, `%`, "&PCT;", -1)
	re := regexp.MustCompile("[^\x00-\x7F]+")
	sData = re.ReplaceAllString(sData, "")
	return
}

func (e *Endpoint) onConvertLinks(message string) (messageFixed string) {
	prefix := e.config.ItemURL
	messageFixed = message
	if strings.Count(message, "") > 1 {
		sets := strings.SplitN(message, "", 3)

		itemid, err := strconv.ParseInt(sets[1][0:6], 16, 32)
		if err != nil {
			itemid = 0
		}
		itemname := sets[1][56:]
		itemlink := prefix
		if itemid > 0 && len(prefix) > 0 {
			itemlink = fmt.Sprintf(" %s%d (%s)", itemlink, itemid, itemname)
		} else {
			itemlink = fmt.Sprintf(" *%s* ", itemname)
		}
		messageFixed = sets[0] + itemlink + sets[2]
		if strings.Count(message, "") > 1 {
			messageFixed = e.onConvertLinks(messageFixed)
		}
	}
	return
}
