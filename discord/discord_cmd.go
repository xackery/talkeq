package discord

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/xackery/talkeq/tlog"
)

func (t *Discord) handleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	t.mu.Lock()
	defer t.mu.Unlock()

	cmd := i.ApplicationCommandData().Name
	tlog.Debugf("[discord] command requested: %s", cmd)

	var content string
	var err error
	cmdFunc, ok := t.commands[strings.ToLower(cmd)]
	if ok {
		content, err = cmdFunc(s, i)
	} else {
		err = fmt.Errorf("unknown command")
	}

	if err != nil {
		tlog.Errorf("[discord] run command failed: %s", err)
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   1 << 6,
		},
	})
	if err != nil {
		tlog.Errorf("[discord] interactionRespond failed: %s", err)
	}
}
