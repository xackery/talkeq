package eqlog

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"regexp"
	"sync"

	"github.com/xackery/talkeq/request"
	"github.com/xackery/talkeq/tlog"

	"github.com/pkg/errors"

	"github.com/hpcloud/tail"
	"github.com/xackery/talkeq/config"
)

// EQLog represents a eqlog connection
type EQLog struct {
	ctx         context.Context
	cancel      context.CancelFunc
	isConnected bool
	mutex       sync.RWMutex
	config      config.EQLog
	subscribers []func(interface{}) error
}

// New creates a new eqlog connect
func New(ctx context.Context, config config.EQLog) (*EQLog, error) {
	ctx, cancel := context.WithCancel(ctx)
	t := &EQLog{
		ctx:    ctx,
		config: config,
		cancel: cancel,
	}
	t.mutex.Lock()
	defer t.mutex.Unlock()

	tlog.Debugf("[eqlog] verifying configuration")

	if !config.IsEnabled {
		return t, nil
	}

	if t.config.Path == "" {
		return nil, fmt.Errorf("path must be set")
	}

	_, err := os.Stat(t.config.Path)
	if err != nil {
		return nil, errors.Wrapf(err, "%s", t.config.Path)
	}
	return t, nil
}

// IsConnected returns if a connection is established
func (t *EQLog) IsConnected() bool {
	t.mutex.RLock()
	isConnected := t.isConnected
	t.mutex.RUnlock()
	return isConnected
}

// Connect establishes a new connection with EQLog
func (t *EQLog) Connect(ctx context.Context) error {
	//var err error
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if !t.config.IsEnabled {
		tlog.Debugf("[eqlog] is disabled, skipping connect")
		return nil
	}

	tlog.Infof("[eqlog] connecting to %s...", t.config.Path)

	t.Disconnect(ctx)

	t.ctx, t.cancel = context.WithCancel(ctx)

	go t.loop(ctx)
	t.isConnected = true
	return nil
}

func (t *EQLog) loop(ctx context.Context) {
	fi, err := os.Stat(t.config.Path)
	if err != nil {
		tlog.Warnf("[eqlog] stat polling failed: %s", err)
		t.Disconnect(ctx)
		return
	}
	cfg := tail.Config{
		Follow:    true,
		MustExist: true,
		Poll:      true,
		Location: &tail.SeekInfo{
			Offset: fi.Size(),
		},
		Logger: tail.DiscardingLogger,
	}

	tailer, err := tail.TailFile(t.config.Path, cfg)
	if err != nil {
		tlog.Warnf("[eqlog] tail attempt failed: %s", err)
		t.Disconnect(ctx)
		return
	}

	for line := range tailer.Lines {
		select {
		case <-t.ctx.Done():
			tlog.Debugf("[eqlog] exiting loop")
			return
		default:
		}

		for routeIndex, route := range t.config.Routes {
			if !route.IsEnabled {
				continue
			}
			pattern, err := regexp.Compile(route.Trigger.Regex)
			if err != nil {
				tlog.Debugf("[eqlog] route %d compile failed: %s", routeIndex, err)
				continue
			}
			matches := pattern.FindAllStringSubmatch(line.Text, -1)
			if len(matches) == 0 {
				continue
			}

			name := ""
			message := ""
			if route.Trigger.MessageIndex >= len(matches[0]) {
				message = matches[0][route.Trigger.MessageIndex]
			}
			if route.Trigger.NameIndex >= len(matches[0]) {
				name = matches[0][route.Trigger.NameIndex]
			}

			buf := new(bytes.Buffer)
			if err := route.MessagePatternTemplate().Execute(buf, struct {
				Name    string
				Message string
			}{
				name,
				message,
			}); err != nil {
				tlog.Warnf("[eqlog] execute route %d: %s", routeIndex, err)
				continue
			}
			switch route.Target {
			case "discord":
				req := request.DiscordSend{
					Ctx:       ctx,
					ChannelID: route.ChannelID,
					Message:   buf.String(),
				}
				for _, s := range t.subscribers {
					err = s(req)
					if err != nil {
						tlog.Warnf("[eqlog->discord subscriber %d] discordSend channelID %s message %s failed: %s", route.ChannelID, req.Message, err)
						continue
					}
					tlog.Infof("[eqlog->discord subscriber %d] message: %s", route.ChannelID, req.Message)
				}
			default:
				tlog.Warnf("[eqlog] unsupported target type: %s", route.Target)
				continue
			}
		}
	}
}

// Disconnect stops a previously started connection with EQLog.
// If called while a connection is not active, returns nil
func (t *EQLog) Disconnect(ctx context.Context) error {
	if !t.config.IsEnabled {
		tlog.Debugf("[eqlog] is disabled, skipping disconnect")
		return nil
	}
	if !t.isConnected {
		tlog.Debugf("[eqlog] is already disconnected, skipping disconnect")
		return nil
	}
	t.cancel()
	t.isConnected = false
	return nil
}

// Send attempts to send a message through EQLog.
func (t *EQLog) Send(ctx context.Context, source string, author string, channelID int, message string, optional string) error {
	return fmt.Errorf("not supported")
}

// Subscribe listens for new events on eqlog
func (t *EQLog) Subscribe(ctx context.Context, onMessage func(interface{}) error) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.subscribers = append(t.subscribers, onMessage)
	return nil
}
