package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/xackery/log"
	"github.com/xackery/talkeq/characterdb"
)

func (t *Discord) whoRegister() error {
	log := log.New()
	log.Debug().Msgf("registering who command")
	_, err := t.conn.ApplicationCommandCreate(t.conn.State.User.ID, t.config.ServerID, &discordgo.ApplicationCommand{
		Name:        "who",
		Description: "get a list of players on server, can filter by zone or name with /who <filter>",
	})
	if err != nil {
		return fmt.Errorf("whoRegister commandCreate: %w", err)
	}
	return nil
}

func (t *Discord) who(s *discordgo.Session, i *discordgo.InteractionCreate) (content string, err error) {
	appCmdData := i.ApplicationCommandData()
	/*	if len(appCmdData.Options) == 0 {
		content = "usage: /who all, /who <name>"
		return
	}*/
	arg := ""
	if len(appCmdData.Options) > 0 {
		arg = fmt.Sprintf("%s", i.ApplicationCommandData().Options[0].Value)
		if arg == "all" {
			arg = ""
		}
	}

	content = characterdb.CharactersOnline(arg)
	return
}
