package sqlreport

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"database/sql"

	//used for database connection
	_ "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"github.com/xackery/talkeq/config"
	"github.com/xackery/talkeq/discord"
)

// SQLReport represents a sqlreport connection
type SQLReport struct {
	ctx            context.Context
	cancel         context.CancelFunc
	isConnected    bool
	mutex          sync.RWMutex
	config         config.SQLReport
	conn           *sql.DB
	subscribers    []func(string, string, int, string, string)
	isInitialState bool
	online         int
	discClient     *discord.Discord
}

// New creates a new sqlreport connect
func New(ctx context.Context, config config.SQLReport, discClient *discord.Discord) (*SQLReport, error) {
	ctx, cancel := context.WithCancel(ctx)
	t := &SQLReport{
		ctx:            ctx,
		config:         config,
		cancel:         cancel,
		isInitialState: true,
		discClient:     discClient,
	}
	t.mutex.Lock()
	defer t.mutex.Unlock()

	log.Debug().Msg("verifying sqlreport configuration")

	if !config.IsEnabled {
		return t, nil
	}

	if config.Host == "" {
		config.Host = "127.0.0.1:3036"
	}

	return t, nil
}

// IsConnected returns if a connection is established
func (t *SQLReport) IsConnected() bool {
	t.mutex.RLock()
	isConnected := t.isConnected
	t.mutex.RUnlock()
	return isConnected
}

// Connect establishes a new connection with SQLReport
func (t *SQLReport) Connect(ctx context.Context) error {
	var err error
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if !t.config.IsEnabled {
		log.Debug().Msg("sqlreport is disabled, skipping connect")
		return nil
	}
	log.Info().Msgf("connecting to sqlreport %s...", t.config.Host)

	if t.conn != nil {
		t.conn.Close()
		t.conn = nil
	}
	t.ctx, t.cancel = context.WithCancel(ctx)

	t.conn, err = sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s", t.config.Username, t.config.Password, t.config.Host, t.config.Database))
	if err != nil {
		return errors.Wrap(err, "sqlreport connect")
	}

	go t.loop(ctx)
	t.isConnected = true
	return nil
}

func (t *SQLReport) loop(ctx context.Context) {
	var value int64
	nextReport := 1 * time.Second

	for {
		log.Debug().Msgf("sqlreport sleeping for %0.1fs", nextReport.Seconds())
		select {
		case <-t.ctx.Done():
			log.Debug().Msg("sqlreport exiting loop")
			return
		case <-time.After(nextReport):
		}
		nextReport = 30 * time.Second
		log.Debug().Msg("sqlreport executing")
		t.mutex.Lock()
		for _, e := range t.config.Entries {
			if e.NextReport.After(time.Now()) {
				continue
			}

			r := t.conn.QueryRow(e.Query)
			if err := r.Scan(&value); err != nil {
				log.Warn().Err(err).Msgf("sqlreport query %s", e.Query)
				e.NextReport = time.Now().Add(e.RefreshDuration)
				if nextReport > e.RefreshDuration {
					nextReport = e.RefreshDuration
				}
				continue
			}

			buf := new(bytes.Buffer)
			if err := e.PatternTemplate.Execute(buf, struct {
				Data int64
			}{
				value,
			}); err != nil {
				log.Warn().Err(err).Msgf("sqlreport execute %s", e.Query)
				e.NextReport = time.Now().Add(e.RefreshDuration)
				if nextReport > e.RefreshDuration {
					nextReport = e.RefreshDuration
				}
				continue
			}
			if err := t.discClient.SetChannelName(e.ChannelID, buf.String()); err != nil {
				log.Warn().Err(err).Msgf("sqlreport setchannelname %s", e.Query)
				e.NextReport = time.Now().Add(e.RefreshDuration)
				if nextReport > e.RefreshDuration {
					nextReport = e.RefreshDuration
				}
				continue
			}

			e.NextReport = time.Now().Add(e.RefreshDuration)
			if nextReport > e.RefreshDuration {
				nextReport = e.RefreshDuration
			}
		}
		t.mutex.Unlock()
	}
}

// Disconnect stops a previously started connection with SQLReport.
// If called while a connection is not active, returns nil
func (t *SQLReport) Disconnect(ctx context.Context) error {
	if !t.config.IsEnabled {
		log.Debug().Msg("sqlreport is disabled, skipping disconnect")
		return nil
	}
	if !t.isConnected {
		log.Debug().Msg("sqlreport is already disconnected, skipping disconnect")
		return nil
	}
	t.conn.Close()

	t.cancel()
	t.conn = nil
	t.isConnected = false

	return nil
}

// Send attempts to send a message through SQLReport.
func (t *SQLReport) Send(ctx context.Context, source string, author string, channelID int, message string, optional string) error {
	return fmt.Errorf("SQL reporting does not support send")
}

// Subscribe listens for new events on sqlreport
func (t *SQLReport) Subscribe(ctx context.Context, onMessage func(source string, author string, channelID int, message string, optional string)) error {
	return fmt.Errorf("SQL reporting does not support subscribe")
}
