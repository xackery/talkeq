package telnet

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xackery/log"

	"github.com/pkg/errors"
	"github.com/xackery/talkeq/config"
	"github.com/xackery/talkeq/request"
	"github.com/ziutek/telnet"
)

var (
	playersOnlineRegex = regexp.MustCompile("([0-9]+) players online")
	oldItemLink        = regexp.MustCompile("\\x12([0-9A-Z]{6})[0-9A-Z]{39}([A-Za-z-'`.,!? ]+)\\x12")
	newItemLink        = regexp.MustCompile("\\x12([0-9A-Z]{6})[0-9A-Z]{50}([A-Za-z-'`.,!? ]+)\\x12")
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
	mutex          sync.RWMutex
	config         config.Telnet
	conn           *telnet.Conn
	subscribers    []func(interface{}) error
	isNewTelnet    bool
	isInitialState bool
	online         int
	onlineUsers    []string
}

// New creates a new telnet connect
func New(ctx context.Context, config config.Telnet) (*Telnet, error) {
	log := log.New()
	ctx, cancel := context.WithCancel(ctx)
	t := &Telnet{
		ctx:            ctx,
		config:         config,
		cancel:         cancel,
		isInitialState: true,
		isNewTelnet:    true,
	}
	t.mutex.Lock()
	defer t.mutex.Unlock()

	log.Debug().Msg("verifying telnet configuration")

	if !config.IsEnabled {
		return t, nil
	}
	if config.IsLegacy {
		t.isNewTelnet = false
	}

	if config.Host == "" {
		config.Host = "127.0.0.1:23"
	}

	if config.MessageDeadlineDuration().Seconds() < 1 {
		config.MessageDeadline = "10s"
		//return nil, fmt.Errorf("telnet.message_deadline must be greater than 1s")
	}

	return t, nil
}

// IsConnected returns if a connection is established
func (t *Telnet) IsConnected() bool {
	t.mutex.RLock()
	isConnected := t.isConnected
	t.mutex.RUnlock()
	return isConnected
}

// Connect establishes a new connection with Telnet
func (t *Telnet) Connect(ctx context.Context) error {
	log := log.New()
	var err error
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if !t.config.IsEnabled {
		log.Debug().Msg("telnet is disabled, skipping connect")
		return nil
	}
	log.Info().Msgf("connecting to telnet %s...", t.config.Host)

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
				log.Warn().Err(err).Int("route", routeIndex).Msg("[telnet] execute")
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
					log.Warn().Err(err).Str("channelID", route.ChannelID).Str("message", req.Message).Msg("[telnet->discord]")
					continue
				}
				log.Info().Str("channelID", route.ChannelID).Str("message", req.Message).Msg("[telnet->discord]")
			}
		}

	}

	log.Info().Msg("telnet connected successfully, listening for messages")
	return nil
}

func (t *Telnet) loop(ctx context.Context) {
	log := log.New()
	data := []byte{}
	var err error
	var msg string

	for {
		select {
		case <-t.ctx.Done():
			log.Debug().Msg("exiting telnet loop")
			return
		default:
		}

		data, err = t.conn.ReadUntil("\n")
		if err != nil {
			log.Warn().Err(err).Msgf("telnet read")
			t.Disconnect(context.Background())
			return
		}
		msg = string(data)

		if len(msg) < 3 { //ignore small messages
			continue
		}

		log.Debug().Str("msg", msg).Msg("raw telnet echo")
		t.parsePlayersOnline(msg)

		msg = t.convertLinks(msg)
		for routeIndex, route := range t.config.Routes {
			if route.Trigger.Custom != "" {
				continue
			}
			pattern, err := regexp.Compile(route.Trigger.Regex)
			if err != nil {
				log.Debug().Err(err).Int("route", routeIndex).Msg("compile")
				continue
			}
			matches := pattern.FindAllStringSubmatch(msg, -1)
			if len(matches) == 0 {
				continue
			}

			name := ""
			message := ""
			if route.Trigger.MessageIndex > len(matches[0]) {
				log.Warn().Int("route", routeIndex).Msgf("[telnet] trigger message_index %d greater than matches %d", route.Trigger.MessageIndex, len(matches[0]))
				continue
			}
			message = matches[0][route.Trigger.MessageIndex]
			if route.Trigger.NameIndex > len(matches[0]) {
				log.Warn().Int("route", routeIndex).Msgf("[telnet] name_index %d greater than matches %d", route.Trigger.MessageIndex, len(matches[0]))
				continue
			}
			name = matches[0][route.Trigger.NameIndex]

			buf := new(bytes.Buffer)
			if err := route.MessagePatternTemplate().Execute(buf, struct {
				Name    string
				Message string
			}{
				name,
				message,
			}); err != nil {
				log.Warn().Err(err).Int("route", routeIndex).Msg("[discord] execute")
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
						log.Warn().Err(err).Str("channelID", route.ChannelID).Str("message", req.Message).Msg("[telnet->discord]")
						continue
					}
					log.Info().Str("channelID", route.ChannelID).Str("message", req.Message).Msg("[telnet->discord]")
				}
			default:
				log.Warn().Msgf("unsupported target type: %s", route.Target)
				continue
			}
		}
	}
}

// Who returns number of online players
func (t *Telnet) Who(ctx context.Context) (int, error) {
	err := t.sendLn("who")
	if err != nil {
		return 0, errors.Wrap(err, "who request")
	}
	time.Sleep(100 * time.Millisecond)
	t.mutex.RLock()
	online := t.online
	t.mutex.RUnlock()
	return online, nil
}

// Disconnect stops a previously started connection with Telnet.
// If called while a connection is not active, returns nil
func (t *Telnet) Disconnect(ctx context.Context) error {
	log := log.New()
	if !t.config.IsEnabled {
		log.Debug().Msg("telnet is disabled, skipping disconnect")
		return nil
	}
	if !t.isConnected {
		log.Debug().Msg("telnet is already disconnected, skipping disconnect")
		return nil
	}
	err := t.conn.Close()
	if err != nil {
		log.Warn().Err(err).Msg("telnet disconnect")
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
				log.Warn().Err(err).Int("route", routeIndex).Msg("[telnet] execute")
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
			for _, s := range t.subscribers {
				err = s(req)
				if err != nil {
					log.Warn().Err(err).Str("channelID", route.ChannelID).Str("message", req.Message).Msg("[telnet->discord]")
					continue
				}
				log.Info().Str("channelID", route.ChannelID).Str("message", req.Message).Msg("[telnet->discord]")
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

func (t *Telnet) convertLinks(message string) string {

	matches := newItemLink.FindAllStringSubmatchIndex(message, -1)
	if len(matches) == 0 {
		matches = oldItemLink.FindAllStringSubmatchIndex(message, -1)
	}
	out := message
	for _, submatches := range matches {
		if len(submatches) < 6 {
			continue
		}
		itemLink := message[submatches[2]:submatches[3]]

		itemID, err := strconv.ParseInt(itemLink, 16, 32)
		if err != nil {
		}
		itemName := message[submatches[4]:submatches[5]]

		out = message[0:submatches[0]]
		if itemID > 0 && len(t.config.ItemURL) > 0 {
			out += fmt.Sprintf("%s%d (%s)", t.config.ItemURL, itemID, itemName)
		} else {
			out += fmt.Sprintf("*%s* ", itemName)
		}
		out += message[submatches[1]:]
		out = strings.TrimSpace(out)
		out = t.convertLinks(out)
		break
	}
	return out
}

// alphanumeric sanitizes incoming data to only be valid
func alphanumeric(data string) string {
	re := regexp.MustCompile("[^a-zA-Z0-9_]+")
	data = re.ReplaceAllString(data, "")
	return data
}

func (t *Telnet) parsePlayersOnline(msg string) {
	log := log.New()

	matches := playersOnlineRegex.FindAllStringSubmatch(msg, -1)
	if len(matches) == 0 { //pattern has no match, unsupported emote
		return
	}
	log.Debug().Msg("detected players online pattern")

	if len(matches[0]) < 2 {
		log.Debug().Str("msg", msg).Msg("ignored, no submatch for players online")
		return
	}

	online, err := strconv.Atoi(matches[0][1])
	if err != nil {
		log.Debug().Str("msg", msg).Msg("online count ignored, parse failed")
		return
	}

	t.mutex.Lock()
	t.online = online
	t.onlineUsers = []string{}
	fmt.Println(msg)
	lines := strings.Split(msg, "\n")
	for _, line := range lines {
		if strings.Contains(line, "players online") {
			continue
		}
		t.onlineUsers = append(t.onlineUsers, line)
	}
	t.mutex.Unlock()
	log.Debug().Int("online", online).Msg("updated online count")
}

// WhoCache responds with the latest known who results based on telnet querying
func (t *Telnet) WhoCache(ctx context.Context, search string) string {

	t.mutex.Lock()
	defer t.mutex.Unlock()
	resp := ""

	counter := 0
	for _, user := range t.onlineUsers {
		if !strings.Contains(user, search) {
			continue
		}
		resp += fmt.Sprintf("%s\n", user)
		counter++
	}

	if counter > 0 {
		resp = fmt.Sprintf("There are %d players who match '%s':\n%s", counter, search, resp)
	}
	return resp
}
