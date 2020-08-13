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
	"github.com/xackery/talkeq/channel"
	"github.com/xackery/talkeq/config"
	"github.com/ziutek/telnet"
)

var (
	playersOnlineRegex = regexp.MustCompile("([0-9]+) players online")
	oldItemLink        = regexp.MustCompile("\\x12([0-9A-Z]{6})[0-9A-Z]{39}([A-Za-z'`.,!? ]+)\\x12")
	newItemLink        = regexp.MustCompile("\\x12([0-9A-Z]{6})[0-9A-Z]{50}([A-Za-z'`.,!? ]+)\\x12")
)

// Telnet represents a telnet connection
type Telnet struct {
	ctx            context.Context
	cancel         context.CancelFunc
	isConnected    bool
	mutex          sync.RWMutex
	config         config.Telnet
	conn           *telnet.Conn
	subscribers    []func(string, string, int, string, string)
	isNewTelnet    bool
	isInitialState bool
	online         int
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

	if config.MessageDeadline.Seconds() < 1 {
		config.MessageDeadline.Duration = 10 * time.Second
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
		for _, s := range t.subscribers {
			s("telnet", "Admin", channel.ToInt(channel.OOC), "Server is now UP", "")
		}
	}

	log.Info().Msg("telnet connected successfully, listening for messages")
	return nil
}

func (t *Telnet) loop(ctx context.Context) {
	log := log.New()
	data := []byte{}
	var err error
	author := ""
	channelID := 0
	msg := ""

	pattern := ""
	channels := map[string]int{
		"says ooc,":   260,
		"auctions,":   261,
		"general,":    291,
		"BROADCASTS,": 1001,
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
			t.Disconnect(context.Background())
			return
		}
		msg = string(data)

		if len(msg) < 3 { //ignore small messages
			continue
		}

		channelID = 0
		for k, v := range channels {
			if !strings.Contains(msg, k) {
				continue
			}
			channelID = v
			pattern = k
			break
		}

		log.Debug().Str("msg", msg).Msg("raw telnet echo")
		t.parsePlayersOnline(msg)

		// first, see if a custom telnet entry regex pattern matches
		isMatch := false
		rmsg := t.convertLinks(msg)
		for _, c := range t.config.Entries {
			vreg, err := regexp.Compile(c.Regex)
			if err != nil {
				log.Debug().Str("regex", c.Regex).Msg("invalid regex found for entry")
				continue
			}

			matches := vreg.FindAllStringSubmatch(rmsg, -1)
			if len(matches) == 0 { //pattern has no match, unsupported emote
				continue
			}
			channelID, err = strconv.Atoi(c.ChannelID)
			if err != nil {
				log.Debug().Str("channelid", c.ChannelID).Msg("invalid channel id for atoi")
				continue
			}

			log.Debug().Msgf("found regex pattern match: %s", c.Regex)

			regexGroup1 := ""
			if len(matches[0]) > 1 {
				regexGroup1 = matches[0][1]
			}
			regexGroup2 := ""
			if len(matches[0]) > 2 {
				regexGroup1 = matches[0][2]
			}
			regexGroup3 := ""
			if len(matches[0]) > 3 {
				regexGroup1 = matches[0][3]
			}
			regexGroup4 := ""
			if len(matches[0]) > 4 {
				regexGroup1 = matches[0][4]
			}

			finalMsg := ""
			buf := new(bytes.Buffer)
			if err := c.MessagePatternTemplate.Execute(buf, struct {
				Msg           string
				Author        string
				ChannelNumber string
				RegexGroup1   string
				RegexGroup2   string
				RegexGroup3   string
				RegexGroup4   string
			}{
				rmsg,
				author,
				string(channelID),
				regexGroup1,
				regexGroup2,
				regexGroup3,
				regexGroup4,
			}); err != nil {
				log.Warn().Err(err).Msgf("telnet execute pattern %s for regex %s", c.MessagePattern, c.Regex)
				continue
			}
			finalMsg = buf.String()

			t.mutex.RLock()
			if len(t.subscribers) == 0 {
				t.mutex.RUnlock()
				log.Debug().Msg("telnet message, but no subscribers to notify, ignoring")
				continue
			}

			for _, s := range t.subscribers {
				s("telnet", author, channelID, finalMsg, "")
			}
			t.mutex.RUnlock()
			isMatch = true
			break
		}
		if isMatch {
			continue
		}

		if channelID == 0 {
			log.Debug().Str("msg", msg).Msg("ignored (unknown channel msg)")
			continue
		}

		//prompt clearing
		if strings.Contains(msg, ">") &&
			strings.Index(msg, ">") < strings.Index(msg, " ") {
			msg = msg[strings.Index(msg, ">")+1:]
		}

		//there's a double user> prompt issue that happens some times, this helps remedy it
		if strings.Contains(msg, ">") &&
			strings.Index(msg, ">") < strings.Index(msg, pattern) {
			msg = msg[strings.Index(msg, ">")+1:]
		}

		//there's a double user> prompt issue that happens some times, this also helps remedy it
		//removed, may be overkill
		//for strings.Contains(msg, "\b") {
		//	msg = msg[strings.Index(msg, "\b")+1:]
		//}
		//just strip any \b text
		msg = strings.ReplaceAll(msg, "\b", "")

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

		msg = msg[strings.Index(msg, pattern)+11 : len(msg)-padOffset]
		if pattern == "general," && strings.Index(msg, "[#General ") > 0 { //general has a special message append
			msg = msg[strings.Index(msg, "[#General ")+10:]
		}
		author = strings.ReplaceAll(author, "_", " ")
		msg = t.convertLinks(msg)
		msg = strings.ReplaceAll(msg, "\n", "")
		t.mutex.RLock()
		if len(t.subscribers) == 0 {
			t.mutex.RUnlock()
			log.Debug().Msg("telnet message, but no subscribers to notify, ignoring")
			continue
		}

		p := 0
		if t.config.IsOOCAuctionEnabled {
			p = strings.Index(msg, "WTS ")
			if p > -1 {
				channelID = channel.ToInt(channel.Auction)
			}
			p = strings.Index(msg, "WTB ")
			if p > -1 {
				channelID = channel.ToInt(channel.Auction)
			}
		}

		for _, s := range t.subscribers {
			s("telnet", author, channelID, msg, "")
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
		for _, s := range t.subscribers {
			s("telnet", "Admin", channel.ToInt(channel.OOC), "Server is now DOWN", "")
		}
	}
	return nil
}

// Send attempts to send a message through Telnet.
func (t *Telnet) Send(ctx context.Context, source string, author string, channelID int, message string, optional string) error {
	log := log.New()
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
func (t *Telnet) Subscribe(ctx context.Context, onMessage func(source string, author string, channelID int, message string, optional string)) error {
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
	t.mutex.Unlock()
	log.Debug().Int("online", online).Msg("updated online count")
}
