package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"sync"

	"github.com/gorilla/mux"
	"github.com/xackery/log"
	"github.com/xackery/talkeq/config"
	"github.com/xackery/talkeq/discord"
	"github.com/xackery/talkeq/registerdb"
	"github.com/xackery/talkeq/request"
)

// API represents the api service
type API struct {
	ctx            context.Context
	cancel         context.CancelFunc
	isConnected    bool
	mutex          sync.RWMutex
	config         config.API
	conn           *sql.DB
	subscribers    []func(interface{}) error
	isInitialState bool
	discord        *discord.Discord
}

const (
	//ActionReply is used when replying to a discord message
	ActionReply = "reply"
)

// New creates a new api endpoint
func New(ctx context.Context, config config.API, discord *discord.Discord) (*API, error) {
	ctx, cancel := context.WithCancel(ctx)
	t := &API{
		ctx:            ctx,
		config:         config,
		cancel:         cancel,
		isInitialState: true,
		discord:        discord,
	}
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if !config.IsEnabled {
		return t, nil
	}

	var err error
	if config.APIRegister.IsEnabled {
		err = registerdb.New(&config)
		if err != nil {
			return nil, fmt.Errorf("registerdb.New: %w", err)
		}

	}

	return t, nil
}

// Subscribe starts a subscription listening on specified data
func (t *API) Subscribe(ctx context.Context, onMessage func(interface{}) error) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.subscribers = append(t.subscribers, onMessage)
	return nil
}

// Command sends a API command
func (t *API) Command(req request.APICommand) error {
	ctx := req.Ctx
	log := log.New()
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	if !t.config.IsEnabled {
		return fmt.Errorf("API is not enabled")
	}

	if !t.isConnected {
		return fmt.Errorf("API is not connected")
	}

	if strings.Index(req.Message, "!") != 0 {
		log.Debug().Msgf("ignoring non command")
		return nil
	}
	args := []string{req.Message}
	if strings.Contains(req.Message, " ") {
		args = strings.Split(req.Message, " ")
	}
	if len(args[0]) < 1 {
		log.Debug().Msg("command too short to parse")
		return nil
	}

	switch strings.ToLower(args[0][1:]) {
	case "register":
		if !t.config.APIRegister.IsEnabled {
			log.Debug().Msg("!register command attempted, but ignored (not enabled)")
			return nil
		}

		if len(args) < 2 {
			msg := request.DiscordSend{
				Ctx:       ctx,
				ChannelID: req.FromDiscordChannelID,
				Message:   "usage: `!register <character>`\nThis command will bind your discord account to provided Everquest character. Your messages in discord will be seen in game as this character name.\nTo change your character after registering, simply repeat process.",
			}
			for _, s := range t.subscribers {
				err := s(msg)
				if err != nil {
					log.Warn().Err(err).Msg("[api->discord]")
				}
			}
			return nil
		}
		character := args[1]

		entry, err := registerdb.Entry(req.FromDiscordNameID)
		if err == nil { //existing entry found
			if entry.Status != "Denied" && entry.Timeout >= time.Now().Unix() {
				remainingTime := time.Until(time.Unix(entry.Timeout, 0)).Minutes()
				remainingMsg := fmt.Sprintf("%0.1f minutes", remainingTime)
				if remainingTime < 1 {
					remainingMsg = fmt.Sprintf("%0.1f seconds", time.Until(time.Unix(entry.Timeout, 0)).Seconds())
				}
				if remainingTime > 60 {
					remainingMsg = fmt.Sprintf("%0.1f hours", time.Until(time.Unix(entry.Timeout, 0)).Hours())
				}

				reply := request.DiscordSend{
					Ctx:       ctx,
					ChannelID: req.FromDiscordChannelID,
					Message:   fmt.Sprintf("!register denied: wait %s to change your name", remainingMsg),
				}
				for _, s := range t.subscribers {
					err = s(reply)
					if err != nil {
						log.Warn().Err(err).Msg("[api->discord] reply to !register")
					}
				}
				return nil
			}
		}

		reply := request.DiscordSend{
			Ctx:       ctx,
			ChannelID: req.FromDiscordChannelID,
			Message:   fmt.Sprintf("I sent a /tell to %s, you have 2 minutes to go in game and [ accept ] it. Status: In Queue", character),
		}
		for _, s := range t.subscribers {
			err = s(reply)
			if err != nil {
				log.Warn().Err(err).Msg("[api->discord] reply to !register")
				continue
			}
			log.Info().Str("message", reply.Message).Msg("[api->discord] !register")

		}
		channelID, messageID, err := t.discord.LastSentMessage()
		if err != nil {
			return fmt.Errorf("lastSentMessage: %w", err)
		}
		registerdb.Set(req.FromDiscordNameID, req.FromDiscordName, character, channelID, messageID, "In Queue", time.Now().Add(30*time.Second).Unix())
	}
	return nil
}

// Connect establishes a server for API
func (t *API) Connect(ctx context.Context) error {
	log := log.New()
	var err error
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if !t.config.IsEnabled {
		log.Debug().Msg("api is disabled, skipping connect")
		return nil
	}

	log.Info().Msgf("api listening on %s...", t.config.Host)

	if t.conn != nil {
		t.conn.Close()
		t.conn = nil
		t.cancel()
	}

	t.ctx, t.cancel = context.WithCancel(ctx)
	r := mux.NewRouter()

	r.HandleFunc("/api", t.index).Methods("GET")
	r.HandleFunc("/api/relays", t.relays).Methods("GET")
	r.HandleFunc("/api/register/confirm", t.registerConfirm).Methods("GET")

	// Start server
	go func() {
		err = http.ListenAndServe(t.config.Host, r)
		if err != nil {
			log.Error().Err(err).Msg("api listenandserver")
		}
		t.mutex.Lock()
		t.isConnected = false
		t.mutex.Unlock()
	}()

	t.isConnected = true

	log.Info().Msgf("api started successfully")

	return nil
}

// IsConnected returns if a connection is established
func (t *API) IsConnected() bool {
	t.mutex.RLock()
	isConnected := t.isConnected
	t.mutex.RUnlock()
	return isConnected
}

// Disconnect stops a previously started connection with Discord.
// If called while a connection is not active, returns nil
func (t *API) Disconnect(ctx context.Context) error {
	log := log.New()
	if !t.config.IsEnabled {
		log.Debug().Msg("api is disabled, skipping disconnect")
		return nil
	}
	if !t.isConnected {
		log.Debug().Msg("api is already disconnected, skipping disconnect")
		return nil
	}
	err := t.conn.Close()
	if err != nil {
		log.Warn().Err(err).Msg("api disconnect")
	}
	t.conn = nil
	t.isConnected = false
	return nil
}

func (t *API) index(w http.ResponseWriter, r *http.Request) {
	log := log.New()
	w.Header().Set("Content-Type", "application/json")
	type Resp struct{}
	err := json.NewEncoder(w).Encode(&Resp{})
	if err != nil {
		log.Warn().Err(err).Msg("encode response")
	}
}
