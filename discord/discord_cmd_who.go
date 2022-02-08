package discord

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/xackery/talkeq/character"
)

func (t *Discord) who(s *discordgo.Session, i *discordgo.InteractionCreate) (content string, err error) {
	appCmdData := i.ApplicationCommandData()
	if len(appCmdData.Options) == 0 {
		content = "usage: /who all, /who <name>"
		return
	}
	arg := ""
	if len(appCmdData.Options) > 0 {
		arg = fmt.Sprintf("%s", i.ApplicationCommandData().Options[0].Value)
		if arg == "all" {
			arg = ""
		}
	}

	content = "There are no players online"
	if arg != "" {
		content = fmt.Sprintf("There are not players who match '%s' online", arg)
	}

	counter := 0
	players := character.CharactersOnline()
	for _, user := range players {
		if arg == "" {
			content += fmt.Sprintf("%s\n", user)
			counter++
			continue
		}

		if !strings.Contains(user.Name, arg) {
			continue
		}
		content += fmt.Sprintf("%s\n", user)
		counter++
	}

	if arg == "" {
		content = fmt.Sprintf("There are %d players online:\n%s", counter, content)
		return
	}
	if counter > 0 {
		content = fmt.Sprintf("There are %d players who match '%s':\n%s", counter, arg, content)
	}
	return
}
