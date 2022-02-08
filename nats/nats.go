package nats

import (
	"context"
	"fmt"
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
	guilds         *database.GuildManager
}

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

// Subscribe listens for new events on nats
func (t *Nats) Subscribe(ctx context.Context, onMessage func(interface{}) error) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.subscribers = append(t.subscribers, onMessage)
	return nil
}
