package discord

import (
	"log"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/xackery/talkeq/model"
	"github.com/xackery/talkeq/service"
)

// Discord represents the discord service
type Discord struct {
	Username    string
	Password    string
	Token       string
	Session     *discordgo.Session
	ServerID    string
	Log         *log.Logger
	Switchboard *service.Switchboard
}

// Name returns the name of the service
func (d *Discord) Name() string {
	return "discord"
}

// Initialize starts a new discord session
func (d *Discord) Initialize() (err error) {
	if d.Log == nil {
		err = errors.New("log not initialized")
		return
	}
	if len(d.ServerID) == 0 {
		err = errors.New("Server ID not configured")
		return
	}

	if len(d.Username) == 0 && len(d.Password) > 0 {
		d.Token = d.Password
	}
	if len(d.Token) > 0 {
		d.Session, err = discordgo.New("Bot " + d.Token)
		if err != nil {
			d.Session = nil
			err = errors.Wrapf(model.ErrAuth{}, "failed to connect with token: %s", err.Error())
			return
		}
		return
	}
	if len(d.Username) > 0 && len(d.Password) > 0 {
		d.Session, err = discordgo.New(d.Username, d.Password)
		if err != nil {
			d.Session = nil
			err = errors.Wrapf(model.ErrAuth{}, "failed to connect with username/password: %s", err.Error())
			return
		}
		return
	}

	if d.Session == nil {
		err = model.ErrAuth{}
		return
	}

	d.Session.StateEnabled = true
	d.Session.AddHandler(d.onMessageEvent)
	err = d.Session.Open()
	if err != nil {
		err = errors.Wrap(err, "failed to open session")
		return
	}
	return
}

// Close will close the discord session.
func (d *Discord) Close() (err error) {
	err = d.Session.Close()
	d.Session = nil
	return
}

// SendChannelMessage handles message requests
func (d *Discord) SendChannelMessage(message *model.ChannelMessage) (err error) {
	if d.Session == nil {
		err = d.Initialize()
		if err != nil {
			err = errors.Wrap(err, "failed to write message")
			return
		}
	}

	channelID := ""
	_, err = d.Session.ChannelMessageSend(channelID, message.Message)
	if err != nil {
		sendError := err
		err = model.ErrMessage{
			Message: message,
		}
		err = errors.Wrapf(err, "failed to send message: %s", sendError.Error())
		return
	}
	return
}

// SendCommandMessage handles message requests
func (d *Discord) SendCommandMessage(message *model.CommandMessage) (err error) {
	err = errors.New("discord does not support writing commands")
	return
}

func (d *Discord) onMessageEvent(s *discordgo.Session, m *discordgo.MessageCreate) {
	ign := ""
	msg := m.ContentWithMentionsReplaced()
	if len(msg) > 4000 {
		msg = msg[0:4000]
	}
	msg = d.sanitize(msg)
	if len(msg) < 1 {
		return
	}

	member, err := s.GuildMember(d.ServerID, m.Author.ID)
	if err != nil {
		d.Log.Printf("failed to get member: %s (Make sure you have set the bot permissions to see members)", err.Error())
		return
	}

	roles, err := s.GuildRoles(d.ServerID)
	if err != nil {
		d.Log.Printf("failed to get roles: %s (Make sure you have set the bot permissions to see roles)", err.Error())
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
		d.Log.Printf("message received from non-ign tagged account: %s", msg)
		return
	}

	ign = d.sanitize(ign)
	if msg[0:1] == "!" {
		//Parse as command
	} else {
		//Relay as message
		message := &model.ChannelMessage{
			Creator: d.Name(),
			Number:  260, //map
			Message: msg,
			From:    ign,
		}
		d.Log.Printf("message: [%d] %s: %s", message.Number, message.From, message.Message)

		services := d.Switchboard.FindPatch(message.Number, d)
		for _, service := range services {
			err = service.SendChannelMessage(message)
			if err != nil {
				d.Log.Printf("-> %s failed: %s\n", service.Name(), err.Error())
				continue
			}
		}
	}

}

func (d *Discord) sanitize(data string) (sData string) {
	sData = data
	sData = strings.Replace(sData, `%`, "&PCT;", -1)
	re := regexp.MustCompile("[^\x00-\x7F]+")
	sData = re.ReplaceAllString(sData, "")
	return
}
