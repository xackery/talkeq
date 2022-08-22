package discord

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/xackery/log"
	"github.com/xackery/talkeq/guilddb"
	"github.com/xackery/talkeq/request"
	"github.com/xackery/talkeq/userdb"
)

func (t *Discord) handleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	ctx := context.Background()
	log := log.New()
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.subscribers) == 0 {
		log.Debug().Msg("[discord] message, but no subscribers to notify, ignoring")
		return
	}

	ign := ""
	msg := m.ContentWithMentionsReplaced()
	if len(msg) > 4000 {
		msg = msg[0:4000]
	}
	msg = sanitize(msg)
	if len(msg) < 1 {
		log.Debug().Str("original message", m.ContentWithMentionsReplaced()).Msg("[discord] message after sanitize too small, ignoring")
		return
	}

	ign = userdb.Name(m.Author.ID)
	if ign == "" {
		ign = t.GetIGNName(s, m.GuildID, m.Author.ID)
		//disabled this code since it would cache results and remove dynamics
		//if ign != "" { //update users database with newly found ign tag
		//	t.users.Set(m.Author.ID, ign)
		//}
	}

	//ignore bot messages
	if m.Author.ID == t.id {
		log.Debug().Msgf("[discord] bot %s ignored (message: %s)", m.Author.ID, msg)
		return
	}

	ign = sanitize(ign)

	if strings.Index(msg, "!") == 0 {
		req := request.APICommand{
			Ctx:                  ctx,
			FromDiscordName:      m.Author.Username,
			FromDiscordNameID:    m.Author.ID,
			FromDiscordChannelID: m.ChannelID,
			FromDiscordIGN:       ign,
			Message:              msg,
		}
		for _, s := range t.subscribers {
			err := s(req)
			if err != nil {
				log.Warn().Err(err).Msg("[discord->api]")
			}
			log.Info().Str("from", m.Author.Username).Str("message", msg).Msg("[discord->api]")
		}
	}

	if len(ign) == 0 {
		log.Warn().Msg("[discord] ign not found, discarding")
		return
	}
	routes := 0
	for routeIndex, route := range t.config.Routes {
		if !route.IsEnabled {
			continue
		}
		if route.Trigger.ChannelID != m.ChannelID {
			continue
		}

		buf := new(bytes.Buffer)

		if err := route.MessagePatternTemplate().Execute(buf, struct {
			Name      string
			Message   string
			ChannelID string
		}{
			ign,
			msg,
			route.ChannelID,
		}); err != nil {
			log.Warn().Err(err).Int("route", routeIndex).Msg("[discord] execute")
			continue
		}

		routes++
		switch route.Target {
		case "telnet":
			req := request.TelnetSend{
				Ctx:     ctx,
				Message: buf.String(),
			}
			for _, s := range t.subscribers {
				err := s(req)
				if err != nil {
					log.Warn().Err(err).Str("message", req.Message).Int("route", routeIndex).Msg("[discord->telnet]")
					continue
				}
				log.Info().Str("message", req.Message).Int("route", routeIndex).Msg("[discord->telnet]")
			}

		default:
			log.Warn().Int("route", routeIndex).Msgf("[discord] invalid target: %s", route.Target)
		}
	}
	//check if channel is a guild one
	guildID := guilddb.GuildID(m.ChannelID)
	if guildID > 0 {
		routes++

		req := request.TelnetSend{
			Ctx:     ctx,
			Message: fmt.Sprintf("guildsay %s %d %s", ign, guildID, msg),
		}
		for _, s := range t.subscribers {
			err := s(req)
			if err != nil {
				log.Warn().Err(err).Str("message", req.Message).Int("guildID", guildID).Msg("[discord->telnet]")
				continue
			}
			log.Info().Str("message", req.Message).Int("guildID", guildID).Msg("[discord->telnet]")
		}
	}
	if routes == 0 {
		log.Debug().Msg("message discarded, not routes match")
	}
}
