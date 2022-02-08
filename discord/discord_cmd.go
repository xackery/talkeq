package discord

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/xackery/log"
)

func (t *Discord) handleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log := log.New()
	t.mu.Lock()
	defer t.mu.Unlock()

	cmd := i.ApplicationCommandData().Name
	log.Debug().Msgf("got interaction: %s", cmd)

	var content string
	var err error
	cmdFunc, ok := t.commands[strings.ToLower(cmd)]
	if ok {
		content, err = cmdFunc(s, i)
	} else {
		err = fmt.Errorf("unknown command")
	}

	if err != nil {
		log.Error().Err(err).Msgf("failed to run command %s", cmd)
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   1 << 6,
		},
	})
	if err != nil {
		log.Error().Err(err).Msgf("interactionRespond")
	}
}
