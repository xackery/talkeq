package discord

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func (t *Discord) who(s *discordgo.Session, i *discordgo.InteractionCreate) (content string, err error) {
	if len(i.ApplicationCommandData().Options) == 0 {
		content = "usage: /who all, /who <name>"
		return
	}
	arg := ""
	if len(i.ApplicationCommandData().Options) > 0 {
		arg = fmt.Sprintf("%s", i.ApplicationCommandData().Options[0].Value)
		if arg == "all" {
			arg = ""
		}
	}

	content = t.telnet.WhoCache(context.Background(), arg)
	return
}
