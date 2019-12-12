package telnet

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/pkg/errors"
	"github.com/xackery/talkeq/channel"
	"github.com/xackery/talkeq/config"
	"github.com/ziutek/telnet"
)

// Telnet represents a telnet connection
type Telnet struct {
	ctx            context.Context
	cancel         context.CancelFunc
	isConnected    bool
	mutex          sync.RWMutex
	config         config.Telnet
	conn           *telnet.Conn
	subscribers    []func(string, string, int, string)
	isNewTelnet    bool
	isInitialState bool
	online         int
}

// New creates a new telnet connect
func New(ctx context.Context, config config.Telnet) (*Telnet, error) {
	ctx, cancel := context.WithCancel(ctx)
	t := &Telnet{
		ctx:            ctx,
		config:         config,
		cancel:         cancel,
		isInitialState: true,
	}
	t.mutex.Lock()
	defer t.mutex.Unlock()

	log.Debug().Msg("verifying telnet configuration")

	if !config.IsEnabled {
		return t, nil
	}

	if config.Host == "" {
		config.Host = "127.0.0.1:23"
	}

	if config.MessageDeadline.Seconds() < 1 {
		return nil, fmt.Errorf("telnet.message_deadline must be greater than 1s")
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
	err = t.conn.SetReadDeadline(time.Now().Add(t.config.MessageDeadline))
	if err != nil {
		return errors.Wrap(err, "set read deadline")
	}
	err = t.conn.SetWriteDeadline(time.Now().Add(t.config.MessageDeadline))
	if err != nil {
		return errors.Wrap(err, "set write deadline")
	}
	t.isNewTelnet = false
	index := 0
	skipAuth := false
	index, err = t.conn.SkipUntilIndex("Username:", "Connection established from localhost, assuming admin")
	if err != nil {
		return errors.Wrap(err, "unexpected initial handshake")
	}
	if index != 0 {
		skipAuth = true
		t.isNewTelnet = true
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
		for _, s := range t.subscribers {
			s("telnet", "Admin", channel.ToInt(channel.OOC), "Server is now UP")
		}
	}
	return nil
}

func (t *Telnet) loop(ctx context.Context) {
	data := []byte{}
	var err error
	author := ""
	channelID := 0
	msg := ""
	var p int
	var online int64

	pattern := ""
	channels := map[string]int{
		"says ooc,": 260,
		"auctions,": 262,
	}
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
			continue
		}
		msg = string(data)
		if len(msg) < 3 { //ignore small messages
			log.Debug().Str("msg", msg).Msg("ignored (too small)")
			continue
		}
		channelID = 0
		for k, v := range channels {
			if strings.Contains(msg, k) {
				channelID = v
				pattern = k
			}
		}

		p = strings.Index(msg, "players online")
		if p > 0 {
			p = strings.Index(msg, " ")
			if p > 0 {
				if msg[p+1:] == "players online" {
					online, err = strconv.ParseInt(msg[:p], 10, 64)
					if err != nil {
						log.Debug().Str("online", msg[:p]).Str("msg", msg).Msg("online count ignored, parse failed?")
					} else {
						t.mutex.Lock()
						t.online = int(online)
						t.mutex.Unlock()
					}
				}
			}
		}

		if channelID == 0 {
			log.Debug().Str("msg", msg).Msg("ignored (unknown channel msg)")
			continue
		}

		//prompt clearing
		if strings.Index(msg, ">") > 0 &&
			strings.Index(msg, ">") < strings.Index(msg, " ") {
			msg = msg[strings.Index(msg, ">")+1:]
		}

		if msg[0:1] == "*" { //ignore echo backs
			log.Debug().Str("msg", msg).Msg("ignored (* = echo back)")
			continue
		}

		author = msg[0:strings.Index(msg, fmt.Sprintf(" %s", pattern))]

		//newTelnet added some odd garbage, this cleans it
		author = strings.Replace(author, ">", "", -1) //remove duplicate prompts
		author = strings.Replace(author, " ", "", -1) //clean up
		author = alphanumeric(author)

		padOffset := 3
		if t.isNewTelnet { //if new telnet, offset is 2 off.
			padOffset = 2
		}

		msg = msg[strings.Index(msg, pattern)+12 : len(msg)-padOffset]
		author = strings.Replace(author, "_", " ", -1)
		msg = t.convertLinks(msg)

		t.mutex.RLock()
		if len(t.subscribers) == 0 {
			t.mutex.RUnlock()
			log.Debug().Msg("telnet message, but no subscribers to notify, ignoring")
			continue
		}

		for _, s := range t.subscribers {
			s("telnet", author, channelID, msg)
		}
		t.mutex.RUnlock()
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
		for _, s := range t.subscribers {
			s("telnet", "Admin", channel.ToInt(channel.OOC), "Server is now DOWN")
		}
	}
	return nil
}

// Send attempts to send a message through Telnet.
func (t *Telnet) Send(ctx context.Context, source string, author string, channelID int, message string) error {
	channelName := channel.ToString(channelID)
	if channelName == "" {
		return fmt.Errorf("invalid channelID: %d", channelID)
	}

	t.mutex.RLock()
	defer t.mutex.RUnlock()

	if !t.config.IsEnabled {
		log.Warn().Str("author", author).Str("channelName", channelName).Str("message", message).Msgf("telnet is disabled, ignoring message")
	}

	if !t.isConnected {
		log.Warn().Str("author", author).Str("channelName", channelName).Str("message", message).Msgf("telnet is not connected, ignoring message")
		return nil
	}

	chatAction := "says"
	if channelName == channel.Auction {
		chatAction = "auctions"
	}
	err := t.sendLn(fmt.Sprintf("emote world %d %s %s from %s, '%s'", channelID, author, chatAction, source, message))
	if err != nil {
		return errors.Wrap(err, "send")
	}
	return nil
}

// Subscribe listens for new events on telnet
func (t *Telnet) Subscribe(ctx context.Context, onMessage func(source string, author string, channelID int, message string)) error {
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
	prefix := t.config.ItemURL
	if strings.Count(message, "") <= 1 {
		return message
	}
	sets := strings.SplitN(message, "", 3)

	itemid, err := strconv.ParseInt(sets[1][0:6], 16, 32)
	if err != nil {
		itemid = 0
	}
	itemname := sets[1][56:]
	itemlink := prefix
	if itemid > 0 && len(prefix) > 0 {
		itemlink = fmt.Sprintf(" %s%d (%s)", itemlink, itemid, itemname)
	} else {
		itemlink = fmt.Sprintf(" *%s* ", itemname)
	}
	message = sets[0] + itemlink + sets[2]
	if strings.Count(message, "") > 1 {
		message = t.convertLinks(message)
	}
	return message
}

// alphanumeric sanitizes incoming data to only be valid
func alphanumeric(data string) string {
	re := regexp.MustCompile("[^a-zA-Z0-9_]+")
	data = re.ReplaceAllString(data, "")
	return data
}
