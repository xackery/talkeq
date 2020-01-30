package nats

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/rs/zerolog/log"

	nats "github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/xackery/talkeq/channel"
	"github.com/xackery/talkeq/config"
	"github.com/xackery/talkeq/database"
	"github.com/xackery/talkeq/pb"
)

// Nats represents a nats connection
type Nats struct {
	ctx            context.Context
	cancel         context.CancelFunc
	isConnected    bool
	mutex          sync.RWMutex
	config         config.Nats
	conn           *nats.Conn
	subscribers    []func(string, string, int, string, string)
	isInitialState bool
	online         int
	guilds         *database.GuildManager
}

// New creates a new nats connect
func New(ctx context.Context, config config.Nats, guildManager *database.GuildManager) (*Nats, error) {
	ctx, cancel := context.WithCancel(ctx)
	t := &Nats{
		ctx:            ctx,
		config:         config,
		cancel:         cancel,
		isInitialState: true,
		guilds:         guildManager,
	}
	t.mutex.Lock()
	defer t.mutex.Unlock()

	log.Debug().Msg("verifying nats configuration")

	if !config.IsEnabled {
		return t, nil
	}

	if config.Host == "" {
		config.Host = "127.0.0.1:23"
	}

	return t, nil
}

// IsConnected returns if a connection is established
func (t *Nats) IsConnected() bool {
	t.mutex.RLock()
	isConnected := t.isConnected
	t.mutex.RUnlock()
	return isConnected
}

// Connect establishes a new connection with Nats
func (t *Nats) Connect(ctx context.Context) error {
	var err error
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if !t.config.IsEnabled {
		log.Debug().Msg("nats is disabled, skipping connect")
		return nil
	}
	log.Info().Msgf("connecting to nats %s...", t.config.Host)

	if t.conn != nil {
		t.conn.Close()
		t.conn = nil
	}
	t.ctx, t.cancel = context.WithCancel(ctx)

	t.conn, err = nats.Connect(fmt.Sprintf("nats://%s", t.config.Host))
	if err != nil {
		return errors.Wrap(err, "nats connect")
	}

	t.conn.Subscribe("world.>", t.onChannelMessage)
	t.conn.Subscribe("global.admin_message.>", t.onAdminMessage)
	t.isConnected = true

	return nil
}

// Disconnect stops a previously started connection with Nats.
// If called while a connection is not active, returns nil
func (t *Nats) Disconnect(ctx context.Context) error {
	if !t.config.IsEnabled {
		log.Debug().Msg("nats is disabled, skipping disconnect")
		return nil
	}
	if !t.isConnected {
		log.Debug().Msg("nats is already disconnected, skipping disconnect")
		return nil
	}
	err := t.conn.Drain()
	if err != nil {
		log.Warn().Err(err).Msg("nats drain")
	}
	t.conn.Close()

	t.cancel()
	t.conn = nil
	t.isConnected = false

	return nil
}

// Send attempts to send a message through Nats.
func (t *Nats) Send(ctx context.Context, source string, author string, channelID int, message string, optional string) error {
	channelName := channel.ToString(channelID)
	if channelName == "" {
		return fmt.Errorf("invalid channelID: %d", channelID)
	}

	t.mutex.RLock()
	defer t.mutex.RUnlock()
	if !t.isConnected {
		return fmt.Errorf("nats not connected")
	}
	channelMessage := &pb.ChannelMessage{
		IsEmote: true,
		Message: fmt.Sprintf("%s says from discord, '%s'", author, message),
		ChanNum: 260,
		Type:    260,
	}
	msg, err := proto.Marshal(channelMessage)
	if err != nil {
		return errors.Wrap(err, "marshal")
	}
	err = t.conn.Publish("ChannelMessageWorld", msg)
	if err != nil {
		return errors.Wrap(err, "publish")
	}

	return nil
}

func (t *Nats) onChannelMessage(m *nats.Msg) {
	var err error
	channelMessage := new(pb.ChannelMessage)
	err = proto.Unmarshal(m.Data, channelMessage)
	if err != nil {
		log.Warn().Err(err).Msg("nats failed to unmarshal channel message")
		return
	}

	if channelMessage.IsEmote {
		channelMessage.ChanNum = channelMessage.Type
	}

	author := channelMessage.From
	msg := channelMessage.Message
	optional := ""

	log.Debug().Str("msg", msg).Msg("processing nats message")

	/*
		if chanType, ok = guilddbid[int(channelMessage.Guilddbid)]; !ok {
			log.Printf("[NATS] Unknown GuildID: %d with message: %s", channelMessage.Guilddbid, channelMessage.Message)
		}

		if chanType, ok = chans[int(channelMessage.ChanNum)]; !ok {
			log.Printf("[NATS] Unknown channel: %d with message: %s", channelMessage.ChanNum, channelMessage.Message)
		}
	*/

	// this likely can be removed or ignored?
	if strings.Contains(channelMessage.Message, "Summoning you to") { //GM messages are relaying to discord!
		log.Debug().Str("msg", msg).Msg("ignoring gm summon")
		return
	}

	msg = t.convertLinks(msg)

	// since we use a different mapping with nats,
	// this helps translate to the universal style
	channels := map[int32]int{
		5: 260, //shout
		4: 261, //auction
		//"general,":  291,
	}

	channelID, ok := channels[channelMessage.ChanNum]

	//check for guild chat
	if !ok && channelMessage.Guilddbid > 0 {
		channelID = 259
		optional = fmt.Sprintf("%d", channelMessage.Guilddbid)

		guildChannelID := t.guilds.ChannelID(int(channelMessage.Guilddbid))
		if len(guildChannelID) == 0 {
			log.Debug().Str("msg", msg).Int32("guildbid", channelMessage.Guilddbid).Msg("guild not found, ignoring message")
			return
		}
	}

	for _, s := range t.subscribers {
		s("nats", author, channelID, msg, optional)
	}
}

// Subscribe listens for new events on nats
func (t *Nats) Subscribe(ctx context.Context, onMessage func(source string, author string, channelID int, message string, optional string)) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.subscribers = append(t.subscribers, onMessage)
	return nil
}

func sanitize(data string) string {
	data = strings.Replace(data, `%`, "&PCT;", -1)
	re := regexp.MustCompile("[^\x00-\x7F]+")
	data = re.ReplaceAllString(data, "")
	return data
}

// alphanumeric sanitizes incoming data to only be valid
func alphanumeric(data string) string {
	re := regexp.MustCompile("[^a-zA-Z0-9_]+")
	data = re.ReplaceAllString(data, "")
	return data
}

func (t *Nats) onAdminMessage(m *nats.Msg) {
	var err error
	channelMessage := new(pb.ChannelMessage)
	err = proto.Unmarshal(m.Data, channelMessage)
	if err != nil {
		log.Warn().Err(err).Msg("nats failed to unmarshal admin message")
		return
	}

	/*	if _, err = disco.SendMessage(config.Discord.CommandChannelID, fmt.Sprintf("**Admin:** %s", channelMessage.Message)); err != nil {
			log.Printf("[NATS] Error sending admin message (%s) %s", channelMessage.Message, err.Error())
			return
		}

		log.Printf("[NATS] AdminMessage: %s\n", channelMessage.Message)*/
}

func (t *Nats) convertLinks(message string) string {
	prefix := t.config.ItemURL
	if strings.Count(message, "") <= 1 {
		return message
	}
	sets := strings.SplitN(message, "", 3)

	itemid, err := strconv.ParseInt(sets[1][0:6], 16, 32)
	if err != nil {
		itemid = 0
	}
	itemname := sets[1][56:]
	itemlink := prefix
	if itemid > 0 && len(prefix) > 0 {
		itemlink = fmt.Sprintf(" %s%d (%s)", itemlink, itemid, itemname)
	} else {
		itemlink = fmt.Sprintf(" *%s* ", itemname)
	}
	message = sets[0] + itemlink + sets[2]
	if strings.Count(message, "") > 1 {
		message = t.convertLinks(message)
	}
	return message
}
