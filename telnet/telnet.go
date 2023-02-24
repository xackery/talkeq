package telnet

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/xackery/talkeq/characterdb"
	"github.com/xackery/talkeq/config"
	"github.com/xackery/talkeq/request"
	"github.com/xackery/talkeq/tlog"
	"github.com/ziutek/telnet"
)

const (
	// ActionMessage represents a telnet message
	ActionMessage = "message"
)

// Telnet represents a telnet connection
type Telnet struct {
	ctx            context.Context
	cancel         context.CancelFunc
	isConnected    bool
	mu             sync.RWMutex
	config         config.Telnet
	conn           *telnet.Conn
	subscribers    []func(interface{}) error
	isNewTelnet    bool
	isInitialState bool
	isPlayerDump   bool
	lastPlayerDump time.Time
	characters     map[string]*characterdb.Character
}

// New creates a new telnet connect
func New(ctx context.Context, config config.Telnet) (*Telnet, error) {
	ctx, cancel := context.WithCancel(ctx)
	t := &Telnet{
		ctx:            ctx,
		config:         config,
		cancel:         cancel,
		isInitialState: true,
		isNewTelnet:    true,
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	tlog.Debugf("[telnet] verifying configuration")

	if !config.IsEnabled {
		return t, nil
	}
	if config.IsLegacy {
		t.isNewTelnet = false
	}

	if config.Host == "" {
		config.Host = "127.0.0.1:23"
	}

	return t, nil
}

// IsConnected returns if a connection is established
func (t *Telnet) IsConnected() bool {
	t.mu.RLock()
	isConnected := t.isConnected
	t.mu.RUnlock()
	return isConnected
}

// Connect establishes a new connection with Telnet
func (t *Telnet) Connect(ctx context.Context) error {
	var err error
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.config.IsEnabled {
		tlog.Debugf("[telnet] is disabled, skipping connect")
		return nil
	}
	tlog.Infof("[telnet] connecting to %s...", t.config.Host)

	isInitialState := t.isInitialState
	t.isInitialState = false
	if t.conn != nil {
		t.conn.Close()
		t.conn = nil
	}
	t.ctx, t.cancel = context.WithCancel(ctx)

	t.conn, err = telnet.Dial("tcp", t.config.Host)
	if err != nil {
		return errors.Wrap(err, "dial")
	}
	err = t.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	if err != nil {
		return errors.Wrap(err, "set read deadline")
	}
	err = t.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err != nil {
		return errors.Wrap(err, "set write deadline")
	}
	index := 0
	skipAuth := false

	index, err = t.conn.SkipUntilIndex("Username:", "Connection established from localhost, assuming admin")
	if err != nil {
		return errors.Wrap(err, "unexpected initial handshake")
	}
	if index != 0 {
		skipAuth = true
	}

	if !skipAuth {
		if t.config.Username == "" {
			return fmt.Errorf("username/password must be set for older servers")
		}

		err = t.sendLn(t.config.Username)
		if err != nil {
			return errors.Wrap(err, "send username")
		}

		err = t.conn.SkipUntil("Password:")
		if err != nil {
			return errors.Wrap(err, "wait for password prompt")
		}

		err = t.sendLn(t.config.Password)
		if err != nil {
			return errors.Wrap(err, "send password")
		}
	}

	err = t.sendLn("echo off")
	if err != nil {
		return errors.Wrap(err, "echo off")
	}

	err = t.sendLn("acceptmessages on")
	if err != nil {
		return errors.Wrap(err, "acceptmessages on")
	}

	t.conn.SetReadDeadline(time.Time{})
	t.conn.SetWriteDeadline(time.Time{})
	go t.loop(ctx)
	t.isConnected = true

	if !isInitialState && t.config.IsServerAnnounceEnabled && len(t.subscribers) > 0 {
		for routeIndex, route := range t.config.Routes {
			if !route.IsEnabled {
				continue
			}
			buf := new(bytes.Buffer)
			if err := route.MessagePatternTemplate().Execute(buf, struct {
				Name    string
				Message string
			}{
				"",
				"",
			}); err != nil {
				tlog.Warnf("[telnet] execute %d failed: %s", routeIndex, err)
				continue
			}

			if route.Trigger.Custom != "serverup" {
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
					tlog.Warnf("[telnet->discord subscriber %d] channelID %s message %s failed: %w", route.ChannelID, req.Message, err)
					continue
				}
				tlog.Infof("[telnet->discord] channelID %s message: %s", route.ChannelID, req.Message)
			}
		}

	}

	tlog.Infof("[telnet] connected successfully, listening for messages")
	return nil
}

func (t *Telnet) loop(ctx context.Context) {
	var data []byte
	var err error
	var msg string

	for {
		select {
		case <-t.ctx.Done():
			tlog.Debugf("[telnet] exiting telnet loop")
			return
		default:
		}

		data, err = t.conn.ReadUntil("\n")
		if err != nil {
			if strings.Contains(err.Error(), "unknown command:") {
				tlog.Debugf("[telnet] received unknown command, ignoring: %s", data)
				continue
			}
			tlog.Warnf("[telnet] read failed: %s", err)
			t.Disconnect(context.Background())
			return
		}
		msg = string(data)

		if len(msg) < 3 { //ignore small messages
			continue
		}

		tlog.Debugf("[telnet] raw echo: %s", strings.ReplaceAll(strings.ReplaceAll(msg, "\r", ""), "\n", ""))

		if t.parsePlayerEntries(msg) {
			continue
		}
		if t.parsePlayersOnline(msg) {
			continue
		}

		if t.parseMessage(msg) {
			continue
		}

	}
}

// Disconnect stops a previously started connection with Telnet.
// If called while a connection is not active, returns nil
func (t *Telnet) Disconnect(ctx context.Context) error {
	if !t.config.IsEnabled {
		tlog.Debugf("[telnet] is disabled, skipping disconnect")
		return nil
	}
	if !t.isConnected {
		tlog.Debugf("[telnet] already disconnected, skipping disconnect")
		return nil
	}
	err := t.conn.Close()
	if err != nil {
		tlog.Warnf("[telnet] disconnect failed, ignoring: %s", err)
	}
	t.cancel()
	t.conn = nil
	t.isConnected = false
	if !t.isInitialState && t.config.IsServerAnnounceEnabled && len(t.subscribers) > 0 {
		for routeIndex, route := range t.config.Routes {
			buf := new(bytes.Buffer)
			if err := route.MessagePatternTemplate().Execute(buf, struct {
				Name    string
				Message string
			}{
				"",
				"",
			}); err != nil {
				tlog.Warnf("[telnet] execute route %d failed, skipping: %s", routeIndex, err)
				continue
			}

			if route.Trigger.Custom != "serverdown" {
				continue
			}
			req := request.DiscordSend{
				Ctx:       ctx,
				ChannelID: route.ChannelID,
				Message:   buf.String(),
			}
			for i, s := range t.subscribers {
				err = s(req)
				if err != nil {
					tlog.Warnf("[telnet->discord subscriber %d] channelID %s message %s failed: %s", i, route.ChannelID, req.Message)
					continue
				}
				tlog.Infof("[telnet->discord subscriber %d] channelID %s message: %s", i, route.ChannelID, req.Message)
			}
		}
	}
	return nil
}

// Send attempts to send a message through Telnet.
func (t *Telnet) Send(req request.TelnetSend) error {
	if !t.config.IsEnabled {
		return fmt.Errorf("telnet is not enabled")
	}

	if !t.isConnected {
		return fmt.Errorf("telnet is not connected")
	}

	err := t.sendLn(req.Message)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}
	return nil
}

// Subscribe listens for new events on telnet
func (t *Telnet) Subscribe(ctx context.Context, onMessage func(interface{}) error) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.subscribers = append(t.subscribers, onMessage)
	return nil
}

func (t *Telnet) sendLn(s string) (err error) {
	if t.conn == nil {
		return fmt.Errorf("no connection created")
	}
	buf := make([]byte, len(s)+1)
	copy(buf, s)
	buf[len(s)] = '\n'

	_, err = t.conn.Write(buf)
	if err != nil {
		return errors.Wrapf(err, "sendLn: %s", s)
	}
	return
}
