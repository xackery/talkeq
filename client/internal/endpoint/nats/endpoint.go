package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/go-nats"
	"github.com/pkg/errors"
	"github.com/xackery/talkeq/client/internal/manager"
	"github.com/xackery/talkeq/model"
)

var (
	queryTimeout = 3 * time.Second
)

// Endpoint implements the Endpointer interface for Discord
type Endpoint struct {
	ctx         context.Context
	isConnected bool
	config      *model.ConfigEndpoint
	conn        *nats.Conn
	queryChan   chan *model.QueryRequest
	manager     *manager.Manager
}

// New creates a new Discord endpoint
func New(ctx context.Context, manager *manager.Manager) (e *Endpoint, err error) {
	e = &Endpoint{
		queryChan: make(chan *model.QueryRequest),
		manager:   manager,
	}
	go e.pump()
	return
}

//Connect establishes a new discord connection
func (e *Endpoint) Connect(ctx context.Context) (err error) {
	_, err = e.runQuery(ctx, "Connect", nil)
	return
}

func (e *Endpoint) onConnect(ctx context.Context) (err error) {
	if e.isConnected {
		e.onClose(ctx)
	}
	if e.config == nil || e.config.NATS == nil {
		err = fmt.Errorf("no configuration found")
		return
	}
	if len(e.config.NATS.IP) == 0 {
		err = fmt.Errorf("IP must be configured")
		return
	}
	if len(e.config.NATS.Port) == 0 {
		err = fmt.Errorf("Port must be configured")
		return
	}

	e.conn, err = nats.Connect(fmt.Sprintf("nats://%s:%s", e.config.NATS.IP, e.config.NATS.Port))
	if err != nil {
		e.conn, err = nats.Connect(nats.DefaultURL)
		if err != nil {
			err = errors.Wrap(err, "failed to connect")
			return
		}
	}

	_, err = e.conn.Subscribe("world.channel_message.out", e.ChannelMessageRead)
	if err != nil {
		err = errors.Wrap(err, "failed to subscribe to channel message")
		return
	}

	e.isConnected = true
	return
}

//ConfigRead returns endpoint configuration currently set for discord
func (e *Endpoint) ConfigRead(ctx context.Context) (resp *model.ConfigEndpoint, err error) {
	respMsg, err := e.runQuery(ctx, "ConfigRead", nil)
	if err != nil {
		err = errors.Wrap(err, "failed to query")
		return
	}
	resp, ok := respMsg.(*model.ConfigEndpoint)
	if !ok {
		err = errors.Wrap(err, "invalid response type")
		return
	}
	return
}

func (e *Endpoint) onConfigRead(ctx context.Context) (resp *model.ConfigEndpoint, err error) {
	resp = e.config
	return
}

//ConfigUpdate sets an endpoint configuration for discord
func (e *Endpoint) ConfigUpdate(ctx context.Context, req *model.ConfigEndpoint) (err error) {
	_, err = e.runQuery(ctx, "ConfigUpdate", req)
	if err != nil {
		return
	}
	return
}

func (e *Endpoint) onConfigUpdate(ctx context.Context, req *model.ConfigEndpoint) (err error) {
	e.config = req
	return
}

// Close closes the discord connection
func (e *Endpoint) Close(ctx context.Context) {
	_, err := e.runQuery(ctx, "Close", nil)
	if err != nil {
		logger := model.NewLogger(ctx)
		logger.Warn().Err(err).Msg("failed to close discord (ignore)")
	}
	return
}

func (e *Endpoint) onClose(ctx context.Context) (err error) {
	e.isConnected = false
	if e.conn != nil {
		e.conn.Close()
	}
	return
}
