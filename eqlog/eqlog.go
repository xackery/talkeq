package eqlog

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/xackery/talkeq/channel"

	"github.com/pkg/errors"
	"github.com/xackery/log"

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
	subscribers []func(string, string, int, string, string)
	isNewEQLog  bool
}

// New creates a new eqlog connect
func New(ctx context.Context, config config.EQLog) (*EQLog, error) {
	log := log.New()
	ctx, cancel := context.WithCancel(ctx)
	t := &EQLog{
		ctx:    ctx,
		config: config,
		cancel: cancel,
	}
	t.mutex.Lock()
	defer t.mutex.Unlock()

	log.Debug().Msg("verifying eqlog configuration")

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
	log := log.New()
	//var err error
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if !t.config.IsEnabled {
		log.Debug().Msg("eqlog is disabled, skipping connect")
		return nil
	}

	log.Info().Msgf("connecting to eqlog %s...", t.config.Path)

	t.Disconnect(ctx)

	t.ctx, t.cancel = context.WithCancel(ctx)

	go t.loop(ctx)
	t.isConnected = true
	return nil
}

func (t *EQLog) loop(ctx context.Context) {
	log := log.New()
	fi, err := os.Stat(t.config.Path)
	if err != nil {
		log.Warn().Err(err).Msg("eqlog stat polling fail")
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
		log.Warn().Err(err).Msg("eqlog tail attempt")
		t.Disconnect(ctx)
		return
	}
	source := "eqlog"
	author := ""
	message := ""
	channelID := 0

	for line := range tailer.Lines {
		select {
		case <-t.ctx.Done():
			log.Debug().Msg("eqlog exiting loop")
			return
		default:
		}
		author, channelID, message, err = t.parse(line.Text)
		if err != nil {
			log.Warn().Err(err).Msg("eqlog parse")
			continue
		}
		if len(message) < 1 {
			log.Debug().Str("text", line.Text).Msg("ignoring empty message")
			continue
		}

		if len(t.subscribers) == 0 {
			log.Debug().Msg("eqlog message, but no subscribers to notify, ignoring")
			continue
		}

		for _, s := range t.subscribers {
			s(source, author, channelID, message, "")
		}
	}
}

func (t *EQLog) parse(msg string) (author string, channelID int, message string, err error) {
	patterns := map[string]int{
		"says out of character, ": channel.ToInt(channel.OOC),
		"auctions, ":              channel.ToInt(channel.Auction),
		"says to general, ":       channel.ToInt(channel.General),
		"shouts, ":                channel.ToInt(channel.Shout),
		"says to guild, ":         channel.ToInt(channel.Guild),
	}
	var pattern string
	var p int
	for pattern, channelID = range patterns {
		if !strings.Contains(msg, pattern) {
			continue
		}
		p = strings.Index(msg, "]")
		if p < 0 {
			err = fmt.Errorf("no ] on msg, invalid logfile?")
			return
		}
		p = strings.Index(msg, "]") + 2
		msg = msg[p:]

		p = strings.Index(msg, " ")
		if p < 0 {
			err = fmt.Errorf("no space after timestamp")
			return
		}

		author = msg[:p]
		msg = msg[:p]

		p = strings.Index(msg, "'")
		if p < 0 {
			err = fmt.Errorf("no single quote encapsulation of message")
			return
		}
		msg = msg[p+1:]
		p = strings.Index(msg, "'")
		if p < 0 {
			err = fmt.Errorf("no single quote ending of message")
			return
		}
		message = msg[:p]

		if pattern == "says to general, " && t.config.IsGeneralChatAuctionEnabled {
			p = strings.Index(msg, "WTS ")
			if p > -1 {
				channelID = channel.ToInt(channel.Auction)
			}
			p = strings.Index(msg, "WTB ")
			if p > -1 {
				channelID = channel.ToInt(channel.Auction)
			}
		}

		return
	}
	return
}

// Disconnect stops a previously started connection with EQLog.
// If called while a connection is not active, returns nil
func (t *EQLog) Disconnect(ctx context.Context) error {
	log := log.New()
	if !t.config.IsEnabled {
		log.Debug().Msg("eqlog is disabled, skipping disconnect")
		return nil
	}
	if !t.isConnected {
		log.Debug().Msg("eqlog is already disconnected, skipping disconnect")
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
func (t *EQLog) Subscribe(ctx context.Context, onMessage func(source string, author string, channelID int, message string, optional string)) error {
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
