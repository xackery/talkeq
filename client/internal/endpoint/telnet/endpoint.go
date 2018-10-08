package telnet

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/xackery/talkeq/client/internal/manager"
	"github.com/xackery/talkeq/model"
	"github.com/ziutek/telnet"
)

var (
	queryTimeout = 3 * time.Second
)

// Endpoint implements the Endpointer interface for Discord
type Endpoint struct {
	ctx         context.Context
	isConnected bool
	config      *model.ConfigEndpoint
	conn        *telnet.Conn
	queryChan   chan *model.QueryRequest
	//telnet changed on eqemu at one point, this detects that change
	isNewTelnet bool
	manager     *manager.Manager
}

// New creates a new Discord endpoint
func New(ctx context.Context, manager *manager.Manager) (e *Endpoint, err error) {
	e = &Endpoint{
		manager:   manager,
		queryChan: make(chan *model.QueryRequest),
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
	if e.config == nil || e.config.Telnet == nil {
		err = fmt.Errorf("no configuration found")
		return
	}
	if len(e.config.Telnet.IP) == 0 {
		err = fmt.Errorf("IP must be configured")
		return
	}
	if len(e.config.Telnet.Port) == 0 {
		err = fmt.Errorf("Port must be configured")
		return
	}

	//First try to connect automatically
	e.conn, err = telnet.Dial("tcp", fmt.Sprintf("%s:%s", e.config.Telnet.IP, e.config.Telnet.Port))
	if err != nil {
		err = errors.Wrap(err, "failed to connect")
		return
	}
	e.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	e.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	e.isNewTelnet = false
	index := 0
	skipAuth := false
	index, err = e.conn.SkipUntilIndex("Username:", "Connection established from localhost, assuming admin")
	if err != nil {
		err = errors.Wrap(err, "failed to establish connection, unexpected initial handshake")
		return
	}
	if index != 0 {
		skipAuth = true
		e.isNewTelnet = true
	}

	if !skipAuth {
		if e.config.Telnet.Username == "" {
			err = fmt.Errorf("username/password must be set for older servers")
			return
		}
		err = e.onSendln(e.config.Telnet.Username)
		if err != nil {
			err = errors.Wrap(err, "failed to send username")
			return
		}

		err = e.conn.SkipUntil("Password:")
		if err != nil {
			err = errors.Wrap(err, "failed to wait for password prompt")
			return
		}

		err = e.onSendln(e.config.Telnet.Password)
		if err != nil {
			err = errors.Wrap(err, "failed to send password")
			return
		}
	}

	err = e.onSendln("echo off")
	if err != nil {
		err = errors.Wrap(err, "failed to send echo off")
		return
	}

	err = e.onSendln("acceptmessages on")
	if err != nil {
		err = errors.Wrap(err, "failed to send acceptmessages on")
		return
	}

	e.conn.SetReadDeadline(time.Time{})
	e.conn.SetWriteDeadline(time.Time{})
	go e.read()

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
		err = e.conn.Close()
		if err != nil {
			return
		}
	}
	return
}

func (e *Endpoint) onSendln(s string) (err error) {
	if !e.isConnected {
		err = errors.Wrap(err, "not connected")
		return
	}
	buf := make([]byte, len(s)+1)
	copy(buf, s)
	buf[len(s)] = '\n'

	_, err = e.conn.Write(buf)
	if err != nil {
		err = errors.Wrapf(err, "failed to write telnet message: %s", s)
		return
	}
	return
}
