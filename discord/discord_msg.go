package discord

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/xackery/talkeq/guilddb"
	"github.com/xackery/talkeq/request"
	"github.com/xackery/talkeq/tlog"
	"github.com/xackery/talkeq/userdb"
)

func (t *Discord) handleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	ctx := context.Background()
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.subscribers) == 0 {
		tlog.Debugf("[discord] message, but no subscribers to notify, ignoring")
		return
	}

	ign := ""

	originalMessage, err := m.ContentWithMoreMentionsReplaced(s)
	if err != nil {
		tlog.Debugf("[discord] message grab failed: %s", err)
		return
	}
	msg := originalMessage
	if len(msg) < 1 {
		tlog.Debugf("[discord] message too small, ignoring, original message: %s", originalMessage)
		return
	}
	if len(msg) > 4000 {
		msg = msg[0:4000]
	}
	msg = sanitize(msg)
	if len(msg) < 1 {
		tlog.Debugf("[discord] message after sanitize too small, ignoring, original message: %s", originalMessage)
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
		tlog.Debugf("[discord] bot %s ignored (message: %s)", m.Author.ID, msg)
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
		for i, s := range t.subscribers {
			err := s(req)
			if err != nil {
				tlog.Warnf("[discord->subscriber %d] request failed: %s", i, err)
			}
			tlog.Infof("[discord->subscriber %d] from %s: %s", m.Author.Username, msg)
		}
	}

	isUnregisteredIGN := false
	if len(ign) == 0 {
		isUnregisteredIGN = true
		for _, route := range t.config.Routes {
			if !route.IsEnabled {
				continue
			}
			if route.Trigger.ChannelID != m.ChannelID {
				continue
			}
			if !route.IsAnyoneAllowed {
				continue
			}
			member, err := s.GuildMember(m.GuildID, m.Author.ID)
			if err != nil {
				tlog.Warnf("[discord] guildMember failed for server_id %s, author_id %s: %s", m.GuildID, m.Author, err)
				continue
			}

			if len(ign) == 0 {
				ign = sanitize(member.Nick)
				if len(ign) == 0 {
					ign = sanitize(member.User.Username)
				}
			}
			tlog.Debugf("[discord] ign not found, but anyone is allowed, using %s", ign)
		}
		if len(ign) == 0 {
			tlog.Warn("[discord] ign not found, discarding")
			return
		}
	}
	routes := 0
	for routeIndex, route := range t.config.Routes {
		if !route.IsEnabled {
			continue
		}
		if route.Trigger.ChannelID != m.ChannelID {
			continue
		}
		if isUnregisteredIGN && !route.IsAnyoneAllowed {
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
			tlog.Warnf("[discord] execute route %d failed: %s", routeIndex, err)
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
					tlog.Warnf("[discord->telnet] route %d message '%s' failed: %s", routeIndex, req.Message, err)
					continue
				}
				tlog.Infof("[discord->telnet] route %d: %s", routeIndex, req.Message)
			}

		default:
			tlog.Warnf("[discord] route %d failed: target %s is invalid", routeIndex, route.Target)
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
		for i, s := range t.subscribers {
			err := s(req)
			if err != nil {
				tlog.Warnf("[discord->subscriber %d] guildID %d message %s failed: %s", i, guildID, req.Message, err)
				continue
			}
			tlog.Infof("[discord->subscriber %d] guildID %d message: %s", i, guildID, req.Message)
		}
	}
	if routes == 0 {
		tlog.Debugf("[discord] message discarded, not routes match")
	}
}
