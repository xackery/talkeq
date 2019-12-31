package peqeditorsql

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/xackery/talkeq/channel"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/hpcloud/tail"
	"github.com/xackery/talkeq/config"
)

// PEQEditorSQL represents a peqeditorsql connection
type PEQEditorSQL struct {
	ctx               context.Context
	cancel            context.CancelFunc
	isConnected       bool
	mutex             sync.RWMutex
	config            config.PEQEditorSQL
	subscribers       []func(string, string, int, string)
	isNewPEQEditorSQL bool
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

	log.Debug().Msg("verifying peqeditorsql configuration")

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
		log.Debug().Msg("peqeditorsql is disabled, skipping connect")
		return nil
	}

	log.Info().Msgf("tailing peqeditorsql %s...", t.config.Path)

	t.Disconnect(ctx)

	t.ctx, t.cancel = context.WithCancel(ctx)

	go t.loop(ctx)
	t.isConnected = true
	return nil
}

func (t *PEQEditorSQL) loop(ctx context.Context) {

	tmpl := template.New("filePattern")
	tmpl.Parse(t.config.FilePattern)

	buf := new(bytes.Buffer)
	tmpl.Execute(buf, struct {
		Year  int
		Month string
	}{
		time.Now().Year(),
		time.Now().Format("01"),
	})

	finalPath := fmt.Sprintf("%s/%s", t.config.Path, buf.String())
	log.Debug().Str("finalPath", finalPath).Msg("tailing file")

	fi, err := os.Stat(finalPath)
	if err != nil {
		log.Warn().Err(err).Msg("peqeditorsql stat polling fail")
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

	tailer, err := tail.TailFile(finalPath, cfg)
	if err != nil {
		log.Warn().Err(err).Msg("peqeditorsql tail attempt")
		t.Disconnect(ctx)
		return
	}
	source := "peqeditorsql"
	author := ""
	message := ""
	channelID := 0

	for line := range tailer.Lines {
		select {
		case <-t.ctx.Done():
			log.Debug().Msg("peqeditorsql exiting loop")
			return
		default:
		}
		author, channelID, message, err = t.parse(line.Text)
		if err != nil {
			log.Warn().Err(err).Msg("peqeditorsql parse")
			continue
		}
		if len(message) < 1 {
			log.Debug().Str("text", line.Text).Msg("ignoring empty message")
			continue
		}

		if len(t.subscribers) == 0 {
			log.Debug().Msg("peqeditorsql message, but no subscribers to notify, ignoring")
			continue
		}

		for _, s := range t.subscribers {
			s(source, author, channelID, message)
		}
	}
}

func (t *PEQEditorSQL) parse(msg string) (author string, channelID int, message string, err error) {
	return "", channel.ToInt(channel.PEQEditorSQLLog), msg, nil
}

// Disconnect stops a previously started connection with PEQEditorSQL.
// If called while a connection is not active, returns nil
func (t *PEQEditorSQL) Disconnect(ctx context.Context) error {
	if !t.config.IsEnabled {
		log.Debug().Msg("peqeditorsql is disabled, skipping disconnect")
		return nil
	}
	if !t.isConnected {
		log.Debug().Msg("peqeditorsql is already disconnected, skipping disconnect")
		return nil
	}
	t.cancel()
	t.isConnected = false
	return nil
}

// Send attempts to send a message through PEQEditorSQL.
func (t *PEQEditorSQL) Send(ctx context.Context, source string, author string, channelID int, message string) error {
	return fmt.Errorf("not supported")
}

// Subscribe listens for new events on peqeditorsql
func (t *PEQEditorSQL) Subscribe(ctx context.Context, onMessage func(source string, author string, channelID int, message string)) error {
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
