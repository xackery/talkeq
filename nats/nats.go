package nats

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/xackery/log"

	nats "github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"github.com/xackery/talkeq/config"
	"github.com/xackery/talkeq/database"
	"github.com/xackery/talkeq/pb"
	"github.com/xackery/talkeq/request"
)

const (
	// ActionMessage is used when sending a messge
	ActionMessage = "message"
)

// Nats represents a nats connection
type Nats struct {
	ctx            context.Context
	cancel         context.CancelFunc
	isConnected    bool
	mutex          sync.RWMutex
	config         config.Nats
	conn           *nats.Conn
	subscribers    []func(interface{}) error
	isInitialState bool
	online         int
	guilds         *database.GuildManager
}

var (
	oldItemLink = regexp.MustCompile("\\x12([0-9A-Z]{6})[0-9A-Z]{39}([A-Za-z'`.,!? ]+)\\x12")
	newItemLink = regexp.MustCompile("\\x12([0-9A-Z]{6})[0-9A-Z]{50}([A-Za-z'`.,!? ]+)\\x12")
)

// New creates a new nats connect
func New(ctx context.Context, config config.Nats, guildManager *database.GuildManager) (*Nats, error) {
	log := log.New()
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
	log := log.New()
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
	log := log.New()
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
func (t *Nats) Send(req request.NatsSend) error {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	if !t.isConnected {
		return fmt.Errorf("nats not connected")
	}

	channelMessage := &pb.ChannelMessage{
		IsEmote:   true,
		Message:   req.Message,
		From:      req.From,
		ChanNum:   req.ChannelID,
		Type:      req.ChannelID,
		Guilddbid: req.GuildID,
	}

	msg, err := proto.Marshal(channelMessage)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	err = t.conn.Publish("world.channel_message.in", msg)
	if err != nil {
		return fmt.Errorf("publish: %w", err)
	}
	return nil
}

func (t *Nats) onChannelMessage(m *nats.Msg) {
	log := log.New()
	var err error
	ctx := context.Background()
	channelMessage := new(pb.ChannelMessage)
	err = proto.Unmarshal(m.Data, channelMessage)
	if err != nil {
		log.Warn().Err(err).Msg("nats failed to unmarshal channel message")
		return
	}

	if channelMessage.IsEmote {
		channelMessage.ChanNum = channelMessage.Type
	}

	msg := channelMessage.Message

	log.Debug().Str("msg", msg).Msg("processing nats message")

	// this likely can be removed or ignored?
	if strings.Contains(channelMessage.Message, "Summoning you to") { //GM messages are relaying to discord!
		log.Debug().Str("msg", msg).Msg("ignoring gm summon")
		return
	}

	msg = t.convertLinks(msg)

	for routeIndex, route := range t.config.Routes {
		buf := new(bytes.Buffer)
		if err := route.MessagePatternTemplate().Execute(buf, struct {
			Name    string
			Message string
		}{
			channelMessage.From,
			channelMessage.Message,
		}); err != nil {
			log.Warn().Err(err).Int("route", routeIndex).Msg("[telnet] execute")
			continue
		}

		if route.Trigger.Custom != "" { //custom can be used to channel bind in nats
			routeChannelID, err := strconv.Atoi(route.Trigger.Custom)
			if err != nil {
				continue
			}
			if int(channelMessage.ChanNum) != routeChannelID {
				continue
			}
			req := request.DiscordSend{
				Ctx:       ctx,
				ChannelID: route.ChannelID,
				Message:   buf.String(),
			}
			for _, s := range t.subscribers {
				err = s(req)
				if err != nil {
					log.Warn().Err(err).Msg("[nats->discord]")
				}
			}
			continue
		}

		pattern, err := regexp.Compile(route.Trigger.Regex)
		if err != nil {
			log.Debug().Err(err).Int("route", routeIndex).Msg("[nats] compile")
			continue
		}
		matches := pattern.FindAllStringSubmatch(channelMessage.Message, -1)
		if len(matches) == 0 {
			continue
		}

		//find regex match
		req := request.DiscordSend{
			Ctx:       ctx,
			ChannelID: route.ChannelID,
			Message:   buf.String(),
		}
		for _, s := range t.subscribers {
			err = s(req)
			if err != nil {
				log.Warn().Err(err).Msg("[nats->discord]")
			}
		}
	}

}

// Subscribe listens for new events on nats
func (t *Nats) Subscribe(ctx context.Context, onMessage func(interface{}) error) error {
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
	log := log.New()
	var err error
	ctx := context.Background()
	channelMessage := new(pb.ChannelMessage)
	err = proto.Unmarshal(m.Data, channelMessage)
	if err != nil {
		log.Warn().Err(err).Msg("nats failed to unmarshal admin message")
		return
	}

	for routeIndex, route := range t.config.Routes {
		if route.Trigger.Custom != "admin" {
			continue
		}
		buf := new(bytes.Buffer)
		if err := route.MessagePatternTemplate().Execute(buf, struct {
			Name    string
			Message string
		}{
			channelMessage.From,
			channelMessage.Message,
		}); err != nil {
			log.Warn().Err(err).Int("route", routeIndex).Msg("[nats] execute")
			continue
		}

		req := request.DiscordSend{
			Ctx:       ctx,
			ChannelID: route.ChannelID,
			Message:   buf.String(),
		}
		for _, s := range t.subscribers {
			err = s(req)
			if err != nil {
				log.Warn().Err(err).Msg("[nats->discord]")
			}
		}
	}
}

func (t *Nats) convertLinks(message string) string {

	matches := newItemLink.FindAllStringSubmatchIndex(message, -1)
	if len(matches) == 0 {
		matches = oldItemLink.FindAllStringSubmatchIndex(message, -1)
	}
	out := message
	for _, submatches := range matches {
		if len(submatches) < 6 {
			continue
		}
		itemLink := message[submatches[2]:submatches[3]]

		itemID, err := strconv.ParseInt(itemLink, 16, 32)
		if err != nil {
		}
		itemName := message[submatches[4]:submatches[5]]

		out = message[0:submatches[0]]
		if itemID > 0 && len(t.config.ItemURL) > 0 {
			out += fmt.Sprintf("%s%d (%s)", t.config.ItemURL, itemID, itemName)
		} else {
			out += fmt.Sprintf("*%s* ", itemName)
		}
		out += message[submatches[1]:]
		out = strings.TrimSpace(out)
		out = t.convertLinks(out)
		break
	}
	return out
}
