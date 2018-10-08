package eqlog

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/xackery/talkeq/model"
)

func (e *Endpoint) read() {
	req := &model.ChatMessage{}
	var err error
	ctx := context.Background()

	for line := range e.conn.Lines {
		req = &model.ChatMessage{
			SourceEndpoint: "eqlog",
		}
		req.Message = line.Text
		select {
		case <-e.ctx.Done():
			return
		default:
		}
		if len(req.Message) < 3 {
			continue
		}

		if strings.Contains(req.Message, "says out of character,") {
			req.ChannelNumber = 260
		} else {
			continue
		}

		req.Author = req.Message[0:strings.Index(req.Message, " says out of character,")]
		if strings.Contains(req.Author, "]") {
			req.Author = req.Author[strings.Index(req.Author, "]")+1 : len(req.Author)]
		}

		req.Author = model.Alphanumeric(req.Author)

		req.Message = req.Message[strings.Index(req.Message, "says out of character, '")+24 : len(req.Message)-1]
		req.Author = strings.Replace(req.Author, "_", " ", -1)
		req.Message = e.onConvertLinks(req.Message) //This may work in log poller
		err = e.manager.ChatMessage(ctx, req)
		if err != nil {
			err = errors.Wrap(err, "manager error")
			return
		}
	}
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
