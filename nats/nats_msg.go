package nats

import (
	"bytes"
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/gogo/protobuf/proto"
	nats "github.com/nats-io/nats.go"
	"github.com/xackery/log"
	"github.com/xackery/talkeq/pb"
	"github.com/xackery/talkeq/request"
)

func (t *Nats) onChannelMessage(m *nats.Msg) {
	log := log.New()
	var err error
	ctx := context.Background()
	channelMessage := new(pb.ChannelMessage)
	err = proto.Unmarshal(m.Data, channelMessage)
	if err != nil {
		log.Warn().Err(err).Msg("nats failed to unmarshal channel message")
		return
	}

	if channelMessage.IsEmote {
		channelMessage.ChanNum = channelMessage.Type
	}

	msg := channelMessage.Message

	log.Debug().Str("msg", msg).Msg("processing nats message")

	// this likely can be removed or ignored?
	if strings.Contains(channelMessage.Message, "Summoning you to") { //GM messages are relaying to discord!
		log.Debug().Str("msg", msg).Msg("ignoring gm summon")
		return
	}

	for routeIndex, route := range t.config.Routes {
		buf := new(bytes.Buffer)
		if err := route.MessagePatternTemplate().Execute(buf, struct {
			Name    string
			Message string
		}{
			channelMessage.From,
			channelMessage.Message,
		}); err != nil {
			log.Warn().Err(err).Int("route", routeIndex).Msg("[telnet] execute")
			continue
		}

		if route.Trigger.Custom != "" { //custom can be used to channel bind in nats
			routeChannelID, err := strconv.Atoi(route.Trigger.Custom)
			if err != nil {
				continue
			}
			if int(channelMessage.ChanNum) != routeChannelID {
				continue
			}
			req := request.DiscordSend{
				Ctx:       ctx,
				ChannelID: route.ChannelID,
				Message:   buf.String(),
			}
			for _, s := range t.subscribers {
				err = s(req)
				if err != nil {
					log.Warn().Err(err).Str("channelID", route.ChannelID).Str("message", req.Message).Msg("[nats->discord]")
					continue
				}
				log.Info().Str("channelID", route.ChannelID).Str("message", req.Message).Msg("[nats->discord]")
			}
			continue
		}

		pattern, err := regexp.Compile(route.Trigger.Regex)
		if err != nil {
			log.Debug().Err(err).Int("route", routeIndex).Msg("[nats] compile")
			continue
		}
		matches := pattern.FindAllStringSubmatch(channelMessage.Message, -1)
		if len(matches) == 0 {
			continue
		}

		//find regex match
		req := request.DiscordSend{
			Ctx:       ctx,
			ChannelID: route.ChannelID,
			Message:   buf.String(),
		}
		for _, s := range t.subscribers {
			err = s(req)
			if err != nil {
				log.Warn().Err(err).Str("channelID", route.ChannelID).Str("message", req.Message).Msg("[nats->discord]")
				continue
			}
			log.Info().Str("channelID", route.ChannelID).Str("message", req.Message).Msg("[nats->discord]")
		}
	}

}

func (t *Nats) onAdminMessage(m *nats.Msg) {
	log := log.New()
	var err error
	ctx := context.Background()
	channelMessage := new(pb.ChannelMessage)
	err = proto.Unmarshal(m.Data, channelMessage)
	if err != nil {
		log.Warn().Err(err).Msg("nats failed to unmarshal admin message")
		return
	}

	for routeIndex, route := range t.config.Routes {
		if !route.IsEnabled {
			continue
		}
		if route.Trigger.Custom != "admin" {
			continue
		}
		buf := new(bytes.Buffer)
		if err := route.MessagePatternTemplate().Execute(buf, struct {
			Name    string
			Message string
		}{
			channelMessage.From,
			channelMessage.Message,
		}); err != nil {
			log.Warn().Err(err).Int("route", routeIndex).Msg("[nats] execute")
			continue
		}

		req := request.DiscordSend{
			Ctx:       ctx,
			ChannelID: route.ChannelID,
			Message:   buf.String(),
		}
		for _, s := range t.subscribers {
			err = s(req)
			if err != nil {
				log.Warn().Err(err).Str("channelID", route.ChannelID).Str("message", req.Message).Msg("[nats->discord]")
				continue
			}
			log.Info().Str("channelID", route.ChannelID).Str("message", req.Message).Msg("[nats->discord]")
		}
	}
}
