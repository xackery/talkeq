package peqeditorsql

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/xackery/talkeq/request"
	"github.com/xackery/talkeq/tlog"

	"github.com/hpcloud/tail"
	"github.com/xackery/talkeq/config"
)

const (
	//ActionMessage is used when broadcasting to discord a message
	ActionMessage = "message"
)

// PEQEditorSQL represents a peqeditorsql connection
type PEQEditorSQL struct {
	ctx         context.Context
	cancel      context.CancelFunc
	isConnected bool
	mutex       sync.RWMutex
	config      config.PEQEditorSQL
	subscribers []func(interface{}) error
}

// New creates a new peqeditorsql connect
func New(ctx context.Context, config config.PEQEditorSQL) (*PEQEditorSQL, error) {
	ctx, cancel := context.WithCancel(ctx)
	t := &PEQEditorSQL{
		ctx:    ctx,
		config: config,
		cancel: cancel,
	}
	t.mutex.Lock()
	defer t.mutex.Unlock()

	tlog.Debugf("[peqeditorsql] verifying configuration")

	if !config.IsEnabled {
		return t, nil
	}

	if t.config.Path == "" {
		return nil, fmt.Errorf("path must be set")
	}

	_, err := os.Stat(t.config.Path)
	if err != nil {
		return nil, fmt.Errorf("stat path %s: %w", t.config.Path, err)
	}
	return t, nil
}

// IsConnected returns if a connection is established
func (t *PEQEditorSQL) IsConnected() bool {
	t.mutex.RLock()
	isConnected := t.isConnected
	t.mutex.RUnlock()
	return isConnected
}

// Connect establishes a new connection with PEQEditorSQL
func (t *PEQEditorSQL) Connect(ctx context.Context) error {
	//var err error
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if !t.config.IsEnabled {
		tlog.Debugf("[peqeditorsql] is disabled, skipping connect")
		return nil
	}

	tlog.Infof("[peqeditorsql] tailing %s...", t.config.Path)

	t.Disconnect(ctx)

	t.ctx, t.cancel = context.WithCancel(ctx)

	go t.loop(ctx)
	t.isConnected = true
	return nil
}

func (t *PEQEditorSQL) loop(ctx context.Context) {
	msgChan := make(chan string, 100)
	tail1, err := newTailWatch(t.ctx, &tailReq{
		id:          1,
		filePattern: t.config.FilePattern,
		basePath:    t.config.Path,
		cfg: tail.Config{
			Follow:    true,
			MustExist: false,
			Poll:      true,
			Logger:    tail.DiscardingLogger,
		},
		isNextMonth: false,
	}, msgChan)
	if err != nil {
		tlog.Warnf("[peqeditorsql] tail1 creation failed: %s", err)
		t.Disconnect(ctx)
		return
	}

	err = tail1.restart(msgChan)
	if err != nil {
		tlog.Warnf("[peqeditorsql] tail1 start failed: %s", err)
		t.Disconnect(ctx)
		return
	}

	tail2, err := newTailWatch(t.ctx, &tailReq{
		id:          2,
		filePattern: t.config.FilePattern,
		basePath:    t.config.Path,
		cfg: tail.Config{
			Follow:    true,
			MustExist: false,
			Poll:      true,
			Logger:    tail.DiscardingLogger,
		},
		isNextMonth: true,
	}, msgChan)
	if err != nil {
		tlog.Warnf("[peqeditorsql] tail2 creation failed: %s", err)
		t.Disconnect(ctx)
		return
	}

	err = tail2.restart(msgChan)
	if err != nil {
		tlog.Warnf("[peqeditorsql] tail2 start failed: %s", err)
		t.Disconnect(ctx)
		return
	}

	ticker := time.NewTicker(12 * time.Hour)
	select {
	case <-t.ctx.Done():
		return
	case <-ticker.C:
		tail1.restart(msgChan)
		tail2.restart(msgChan)
	case line := <-msgChan:
		for routeIndex, route := range t.config.Routes {
			if !route.IsEnabled {
				continue
			}
			pattern, err := regexp.Compile(route.Trigger.Regex)
			if err != nil {
				tlog.Debugf("[peqeditorsql] compile route %d skipped: %s", routeIndex, err)
				continue
			}
			matches := pattern.FindAllStringSubmatch(line, -1)
			if len(matches) == 0 {
				continue
			}

			name := ""
			message := ""
			if route.Trigger.MessageIndex > 0 && route.Trigger.MessageIndex <= len(matches[0]) {
				message = matches[0][route.Trigger.MessageIndex]
			}
			if route.Trigger.NameIndex > 0 && route.Trigger.NameIndex <= len(matches[0]) {
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
				tlog.Warnf("[peqeditorsql] execute route %d skipped: %s", routeIndex, err)
				continue
			}
			switch route.Target {
			case "discord":
				req := request.DiscordSend{
					Ctx:       ctx,
					ChannelID: route.ChannelID,
					Message:   buf.String(),
				}
				for i, s := range t.subscribers {
					err = s(req)
					if err != nil {
						tlog.Warnf("[peqeditorsql->discord subscriber %d] channel %s message %s failed: %s", i, route.ChannelID, req.Message)
						continue
					}
					tlog.Infof("[peqeditorsql->discord subscribe %d] channel %s message: %s", i, route.ChannelID, req.Message)
				}
			default:
				tlog.Warnf("[peqeditorsql] unsupported target type: %s", route.Target)
				continue
			}
		}
	}
}

// Disconnect stops a previously started connection with PEQEditorSQL.
// If called while a connection is not active, returns nil
func (t *PEQEditorSQL) Disconnect(ctx context.Context) error {
	if !t.config.IsEnabled {
		tlog.Debugf("[peqeditorsql] is disabled, skipping disconnect")
		return nil
	}
	if !t.isConnected {
		tlog.Debugf("[peqeditorsql] already disconnected, skipping disconnect")
		return nil
	}
	t.cancel()
	t.isConnected = false
	return nil
}

// Send attempts to send a message through PEQEditorSQL.
func (t *PEQEditorSQL) Send(ctx context.Context, source string, author string, channelID int, message string, optional string) error {
	return fmt.Errorf("not supported")
}

// Subscribe listens for new events on peqeditorsql
func (t *PEQEditorSQL) Subscribe(ctx context.Context, onMessage func(interface{}) error) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.subscribers = append(t.subscribers, onMessage)
	return nil
}
