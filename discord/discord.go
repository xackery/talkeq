package discord

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/xackery/talkeq/channel"
	"github.com/xackery/talkeq/config"
)

// Discord represents a discord connection
type Discord struct {
	ctx         context.Context
	cancel      context.CancelFunc
	isConnected bool
	mutex       sync.RWMutex
	config      config.Discord
	conn        *discordgo.Session
	subscribers []func(string, string, int, string)
	id          string
}

// New creates a new discord connect
func New(ctx context.Context, config config.Discord) (*Discord, error) {
	ctx, cancel := context.WithCancel(ctx)

	t := &Discord{
		ctx:    ctx,
		cancel: cancel,
		config: config,
	}
	t.mutex.Lock()
	defer t.mutex.Unlock()

	log.Debug().Msg("verifying discord configuration")

	if !config.IsEnabled {
		return t, nil
	}

	if config.ClientID == "" {
		return nil, fmt.Errorf("client_id must be set")
	}

	if config.Token == "" {
		return nil, fmt.Errorf("bot_token must be set")
	}

	if config.ServerID == "" {
		return nil, fmt.Errorf("server_id must be set")
	}

	return t, nil
}

// Connect establishes a new connection with Discord
func (t *Discord) Connect(ctx context.Context) error {
	var err error
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if !t.config.IsEnabled {
		log.Debug().Msg("discord is disabled, skipping connect")
		return nil
	}

	log.Info().Msgf("discord connecting to server_id %s...", t.config.ServerID)

	if t.conn != nil {
		t.conn.Close()
		t.conn = nil
		t.cancel()
	}
	t.ctx, t.cancel = context.WithCancel(ctx)

	t.conn, err = discordgo.New("Bot " + t.config.Token)
	if err != nil {
		return errors.Wrap(err, "new")
	}

	t.conn.StateEnabled = true
	t.conn.AddHandler(t.handler)

	err = t.conn.Open()
	if err != nil {
		return errors.Wrap(err, "open")
	}

	go t.loop(ctx)

	t.isConnected = true
	if t.config.OOC.ListenChannelID == "" && t.config.Auction.ListenChannelID == "" {
		log.Info().Msgf("discord connected successfully")
		return nil
	}

	listenMsg := "for "

	var st *discordgo.Channel
	chatType := channel.OOC
	if t.config.OOC.ListenChannelID != "" {
		st, err = t.conn.Channel(t.config.OOC.ListenChannelID)
		if err != nil {
			if strings.Contains(err.Error(), "not snowflake") {
				log.Error().Msgf("your bot appears to not be allowed to visit channel %s. visit https://discordapp.com/oauth2/authorize?&client_id=%s&scope=bot&permissions=268504080 and authorize", t.config.OOC.ListenChannelID, t.config.ClientID)
				if runtime.GOOS == "windows" {
					option := ""
					fmt.Println("press a key then enter to exit.")
					fmt.Scan(&option)
				}
				os.Exit(1)
			}
			return errors.Wrapf(err, "find %s channel", chatType)
		}

		listenMsg += "OOC chat in #" + st.Name
	}
	if t.config.Auction.ListenChannelID != "" {
		chatType = channel.Auction
		st, err = t.conn.Channel(t.config.Auction.ListenChannelID)
		if err != nil {
			t.snowflakeCheck(err)
			return errors.Wrapf(err, "find %s channel", chatType)
		}

		if listenMsg != "for " {
			listenMsg += ", "
		}
		listenMsg += "Auction chat in #" + st.Name
	}
	if t.config.General.ListenChannelID != "" {
		chatType = channel.General
		st, err = t.conn.Channel(t.config.General.ListenChannelID)
		if err != nil {
			t.snowflakeCheck(err)
			return errors.Wrapf(err, "find %s channel", chatType)
		}

		if listenMsg != "for " {
			listenMsg += ", "
		}
		listenMsg += "General chat in #" + st.Name
	}
	if t.config.Shout.ListenChannelID != "" {
		chatType = channel.Shout
		st, err = t.conn.Channel(t.config.Shout.ListenChannelID)
		if err != nil {
			t.snowflakeCheck(err)
			return errors.Wrapf(err, "find %s channel", chatType)
		}

		if listenMsg != "for " {
			listenMsg += ", "
		}
		listenMsg += "Shout chat in #" + st.Name
	}
	if t.config.Guild.ListenChannelID != "" {
		chatType = channel.Guild
		st, err = t.conn.Channel(t.config.Guild.ListenChannelID)
		if err != nil {
			t.snowflakeCheck(err)
			return errors.Wrapf(err, "find %s channel", chatType)
		}

		if listenMsg != "for " {
			listenMsg += ", "
		}
		listenMsg += "Guild chat in #" + st.Name
	}

	myUser, err := t.conn.User("@me")
	if err != nil {
		return errors.Wrap(err, "get my username")
	}

	t.id = myUser.ID
	log.Debug().Str("id", t.id).Msg("@me")

	log.Info().Msgf("discord connected successfully, listening %s", listenMsg)

	err = t.StatusUpdate(ctx, 0, "Status: Online")
	if err != nil {
		return err
	}
	return nil
}

func (t *Discord) snowflakeCheck(err error) {
	if !strings.Contains(err.Error(), "not snowflake") {
		return
	}
	log.Error().Msgf("your bot appears to not be allowed to visit channel %s. visit https://discordapp.com/oauth2/authorize?&client_id=%s&scope=bot&permissions=268504080 and authorize", t.config.OOC.ListenChannelID, t.config.ClientID)
	if runtime.GOOS == "windows" {
		option := ""
		fmt.Println("press a key then enter to exit.")
		fmt.Scan(&option)
	}
	os.Exit(1)
}

func (t *Discord) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Debug().Msg("discord loop exit")
			return
		default:
		}

		time.Sleep(60 * time.Second)
	}
}

// StatusUpdate updates the status text on discord
func (t *Discord) StatusUpdate(ctx context.Context, online int, customText string) error {
	var err error
	if customText != "" {
		err = t.conn.UpdateStatus(0, customText)
		if err != nil {
			return err
		}
		return nil
	}
	tmpl := template.New("online")
	tmpl.Parse(t.config.BotStatus)

	buf := new(bytes.Buffer)
	tmpl.Execute(buf, struct {
		PlayerCount int
	}{
		online,
	})

	err = t.conn.UpdateStatus(0, buf.String())
	if err != nil {
		return err
	}
	return nil
}

// IsConnected returns if a connection is established
func (t *Discord) IsConnected() bool {
	t.mutex.RLock()
	isConnected := t.isConnected
	t.mutex.RUnlock()
	return isConnected
}

// Disconnect stops a previously started connection with Discord.
// If called while a connection is not active, returns nil
func (t *Discord) Disconnect(ctx context.Context) error {
	if !t.config.IsEnabled {
		log.Debug().Msg("discord is disabled, skipping disconnect")
		return nil
	}
	if !t.isConnected {
		log.Debug().Msg("discord is already disconnected, skipping disconnect")
		return nil
	}
	err := t.conn.Close()
	if err != nil {
		log.Warn().Err(err).Msg("discord disconnect")
	}
	t.conn = nil
	t.isConnected = false
	return nil
}

// Send attempts to send a message through Discord.
func (t *Discord) Send(ctx context.Context, source string, author string, channelID int, message string) error {
	channelName := channel.ToString(channelID)
	if channelName == "" {
		return fmt.Errorf("invalid channelID: %d", channelID)
	}

	t.mutex.RLock()
	defer t.mutex.RUnlock()

	if !t.config.IsEnabled {
		log.Warn().Str("author", author).Str("channelName", channelName).Str("message", message).Msgf("discord is disabled")
	}

	if !t.isConnected {
		log.Warn().Str("author", author).Str("channelName", channelName).Str("message", message).Msgf("discord is not connected")
		return nil
	}

	sendChannelID := ""
	if channelName == channel.Auction {
		sendChannelID = t.config.Auction.SendChannelID
	}
	if channelName == channel.OOC {
		sendChannelID = t.config.OOC.SendChannelID
	}

	_, err := t.conn.ChannelMessageSend(sendChannelID, fmt.Sprintf("**%s %s:** %s", author, channelName, message))
	if err != nil {
		return errors.Wrapf(err, "send %s %s: %s", author, channelName, message)
	}

	log.Info().Str("author", author).Str("channelName", channelName).Str("message", message).Msg("sent to discord")
	return nil
}

// Subscribe listens for new events on discord
func (t *Discord) Subscribe(ctx context.Context, onMessage func(source string, author string, channelID int, message string)) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.subscribers = append(t.subscribers, onMessage)
	return nil
}

func (t *Discord) handler(s *discordgo.Session, m *discordgo.MessageCreate) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if len(t.subscribers) == 0 {
		log.Debug().Msg("discord message, but no subscribers to notify, ignoring")
		return
	}

	ign := ""
	msg := m.ContentWithMentionsReplaced()
	if len(msg) > 4000 {
		msg = msg[0:4000]
	}
	msg = sanitize(msg)
	if len(msg) < 1 {
		log.Debug().Str("original message", m.ContentWithMentionsReplaced()).Msg("message after sanitize too small, ignoring")
		return
	}

	member, err := s.GuildMember(t.config.ServerID, m.Author.ID)
	if err != nil {
		log.Warn().Err(err).Str("author_id", m.Author.ID).Msg("guild member lookup")
		return
	}

	roles, err := s.GuildRoles(t.config.ServerID)
	if err != nil {
		log.Warn().Err(err).Str("server_id", t.config.ServerID).Msg("get roles")
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
			if strings.TrimSpace(gRole.ID) != strings.TrimSpace(role) {
				continue
			}
			if !strings.Contains(gRole.Name, "IGN:") {
				continue
			}
			splitStr := strings.Split(gRole.Name, "IGN:")
			if len(splitStr) > 1 {
				ign = strings.TrimSpace(splitStr[1])
			}
		}
	}

	if m.Author.ID == t.id {
		log.Debug().Msgf("discord message from bot id %s ignored (message: %s)", m.Author.ID, msg)
		return
	}

	if ign == "" {
		log.Warn().Msgf("discord message from non-IGN tagged account %s ignored (message: %s)", m.Author.Username, msg)
		return
	}

	ign = sanitize(ign)

	channelID := 0
	if t.config.Auction.ListenChannelID == m.ChannelID {
		channelID = channel.ToInt(channel.Auction)
	}
	if t.config.OOC.ListenChannelID == m.ChannelID {
		channelID = channel.ToInt(channel.OOC)
	}

	if channelID == 0 {
		log.Warn().Msgf("discord message from unknown channel %s", m.ChannelID)
		return
	}

	for _, s := range t.subscribers {
		s("discord", ign, channelID, msg)
	}
}

func sanitize(data string) string {
	data = strings.Replace(data, `%`, "&PCT;", -1)
	re := regexp.MustCompile("[^\x00-\x7F]+")
	data = re.ReplaceAllString(data, "")
	return data
}
