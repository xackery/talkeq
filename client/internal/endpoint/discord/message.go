package discord

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/xackery/talkeq/model"
)

// used internally for discord message event handling
type discordMessage struct {
	Session *discordgo.Session
	Message *discordgo.MessageCreate
}

// SendMessage sends a message to discord
func (e *Endpoint) SendMessage(ctx context.Context, req *model.ChatMessage) (err error) {
	return
}

func (e *Endpoint) onSendMessage(ctx context.Context, req *model.ChatMessage) (resp *model.ChatMessage, err error) {
	resp = &model.ChatMessage{}
	if !e.isConnected {
		err = fmt.Errorf("not connected")
		return
	}
	respMsg, err := e.conn.ChannelMessageSend(req.ChannelID, req.Message)
	if err != nil {
		err = errors.Wrap(err, "failed to send message")
		return
	}
	resp.ChannelID = respMsg.ChannelID
	return
}

// ChannelMessageRead consumes messages from discord
func (e *Endpoint) ChannelMessageRead(s *discordgo.Session, m *discordgo.MessageCreate) {
	ctx := context.Background()
	var err error
	_, err = e.runQuery(ctx, "ChannelMessageRead", &discordMessage{Session: s, Message: m})
	if err != nil {
		logger := model.NewLogger(ctx)
		logger.Error().Err(err).Msg("error while reading discord message")
		return
	}
	return
}

func (e *Endpoint) onChannelMessageRead(ctx context.Context, req *discordMessage) (err error) {
	s := req.Session
	m := req.Message
	ign := ""
	msg := m.ContentWithMentionsReplaced()
	if len(msg) > 4000 {
		msg = msg[0:4000]
	}
	msg = e.sanitize(msg)
	if len(msg) < 1 {
		return
	}

	member, err := s.GuildMember(e.config.Discord.ServerID, m.Author.ID)
	if err != nil {
		err = errors.Wrap(err, "failed to get member")
		return
	}

	roles, err := s.GuildRoles(e.config.Discord.ServerID)
	if err != nil {
		err = errors.Wrap(err, "failed to get roles")
		return
	}

	for _, role := range member.Roles {
		if ign != "" {
			break
		}
		for _, gRole := range roles {
			if ign != "" {
				break
			}
			if strings.TrimSpace(gRole.ID) == strings.TrimSpace(role) {
				if strings.Contains(gRole.Name, "IGN:") {
					splitStr := strings.Split(gRole.Name, "IGN:")
					if len(splitStr) > 1 {
						ign = strings.TrimSpace(splitStr[1])
					}
				}
			}
		}
	}

	if ign == "" {
		err = fmt.Errorf("message from non-IGN tagged account ignored: %s", m.Author.Username)
		return
	}

	ign = e.sanitize(ign)
	cReq := &model.ChatMessage{
		Author:        ign,
		ChannelNumber: 260,
		Message:       msg,
	}
	err = e.manager.ChatMessage(ctx, cReq)
	if err != nil {
		err = errors.Wrap(err, "manager error")
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
