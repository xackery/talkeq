package telnet

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/xackery/talkeq/model"
)

func (e *Endpoint) read() {
	data := []byte{}
	var err error
	req := &model.ChatMessage{}
	ctx := context.Background()
	for {
		logger := model.NewLogger(ctx)
		select {
		case <-e.ctx.Done():
			return
		default:
		}
		req = &model.ChatMessage{
			SourceEndpoint: "telnet",
		}

		data, err = e.conn.ReadUntil("\n")
		if err != nil {
			logger.Warn().Err(err).Msg("failed to parse (ignore)")
			continue
		}
		req.Message = string(data)
		if len(req.Message) < 3 { //ignore small messages
			continue
		}

		if strings.Contains(req.Message, "says ooc,") {
			req.ChannelNumber = 260
		} else {
			continue
		}

		//prompt clearing
		if strings.Index(req.Message, ">") > 0 &&
			strings.Index(req.Message, ">") < strings.Index(req.Message, " ") {
			req.Message = req.Message[strings.Index(req.Message, ">")+1:]
		}

		if req.Message[0:1] == "*" { //ignore echo backs
			continue
		}

		req.Author = req.Message[0:strings.Index(req.Message, " says ooc,")]

		//newTelnet added some odd garbage, this cleans it
		req.Author = strings.Replace(req.Author, ">", "", -1) //remove duplicate prompts
		req.Author = strings.Replace(req.Author, " ", "", -1) //clean up
		req.Author = model.Alphanumeric(req.Author)

		padOffset := 3
		if e.isNewTelnet { //if new telnet, offsetis 2 off.
			padOffset = 2
		}
		req.Message = req.Message[strings.Index(req.Message, "says ooc, '")+11 : len(req.Message)-padOffset]
		req.Author = strings.Replace(req.Author, "_", " ", -1)
		req.Message = e.onConvertLinks(req.Message)
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
