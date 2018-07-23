package telnet

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/xackery/talkeq/model"
	"github.com/xackery/talkeq/service"
	"github.com/ziutek/telnet"
)

// Telnet represents the telnet service
type Telnet struct {
	Username    string
	Password    string
	Log         *log.Logger
	Session     *telnet.Conn
	newTelnet   bool
	ItemURL     string
	IP          string
	Port        string
	Switchboard *service.Switchboard
}

// Name returns the name of the service
func (t *Telnet) Name() string {
	return "telnet"
}

// Initialize starts a new telnet session
func (t *Telnet) Initialize() (err error) {
	if t.Log == nil {
		err = errors.New("log not initialized")
		return
	}
	t.newTelnet = false

	//First try to connect automatically
	t.Session, err = telnet.Dial("tcp", fmt.Sprintf("%s:%s", t.IP, t.Port))
	if err != nil {
		t.Session = nil
		err = errors.Wrap(model.ErrAuth{}, err.Error())
		return
	}
	t.Session.SetReadDeadline(time.Now().Add(10 * time.Second))
	t.Session.SetWriteDeadline(time.Now().Add(10 * time.Second))
	index := 0
	skipAuth := false
	if index, err = t.Session.SkipUntilIndex("Username:", "Connection established from localhost, assuming admin"); err != nil {
		t.newTelnet = true
		return
	}
	if index != 0 {
		skipAuth = true
		t.Log.Println("skipping auth")
		t.newTelnet = true
	}

	if !skipAuth {
		if err = t.sendln(t.Username); err != nil {
			return
		}

		if err = t.Session.SkipUntil("Password:"); err != nil {
			return
		}
		if err = t.sendln(t.Password); err != nil {
			return
		}
	}

	if err = t.sendln("echo off"); err != nil {
		return
	}

	if err = t.sendln("acceptmessages on"); err != nil {
		return
	}

	t.Session.SetReadDeadline(time.Time{})
	t.Session.SetWriteDeadline(time.Time{})
	go t.pollMessages()
	return
}

// Close will close the discord session.
func (t *Telnet) Close() (err error) {
	err = t.Session.Close()
	t.Session = nil
	return
}

// WriteMessage handles message requests
func (t *Telnet) WriteMessage(message *model.Message) (err error) {
	if message.ChanNum == 0 {
		message.ChanNum = 260
	}

	err = t.sendln(fmt.Sprintf("emote world %d %s says from discord, '%s'", message.ChanNum, message.From, message.Message))
	if err != nil {
		t.Log.Printf("failed to send telnet message (%s:%s): %s\n", message.From, message.Message, err.Error())
		return
	}
	return
}

func (t *Telnet) pollMessages() {
	data := []byte{}
	message := &model.Message{}
	var err error
	for {
		message.Message = ""
		message.ChanNum = 260
		message.From = ""
		if t.Session == nil {
			return
		}

		data, err = t.Session.ReadUntil("\n")
		if err != nil {
			t.Log.Printf("failed to parse: %s\n", err.Error())
			continue
		}
		message.Message = string(data)
		if len(message.Message) < 3 { //ignore small messages
			continue
		}
		//todo: Add support for new message types
		if !strings.Contains(message.Message, "says ooc,") { //ignore non-ooc
			continue
		}
		if strings.Index(message.Message, ">") > 0 && strings.Index(message.Message, ">") < strings.Index(message.Message, " ") { //ignore prompts
			message.Message = message.Message[strings.Index(message.Message, ">")+1:]
		}
		if message.Message[0:1] == "*" { //ignore echo backs
			continue
		}

		message.From = message.Message[0:strings.Index(message.Message, " says ooc,")]

		//newTelnet added some odd garbage, this cleans it
		message.From = strings.Replace(message.From, ">", "", -1) //remove duplicate prompts
		message.From = strings.Replace(message.From, " ", "", -1) //clean up
		message.From = t.alphanumeric(message.From)

		padOffset := 3
		if t.newTelnet { //if new telnet, offsetis 2 off.
			padOffset = 2
		}
		message.Message = message.Message[strings.Index(message.Message, "says ooc, '")+11 : len(message.Message)-padOffset]
		message.From = strings.Replace(message.From, "_", " ", -1)
		message.Message = t.convertLinks(message.Message)

		//Todo: Send message to writemessage
		t.Log.Printf("message: [%d] %s: %s", message.ChanNum, message.From, message.Message)
		services := t.Switchboard.FindPatch(message.ChanNum, t)
		for _, service := range services {
			err = service.WriteMessage(message)
			if err != nil {
				t.Log.Printf("-> %s failed: %s\n", service.Name(), err.Error())
				continue
			}
		}
	}
}

func (t *Telnet) sendln(s string) (err error) {
	buf := make([]byte, len(s)+1)
	copy(buf, s)
	buf[len(s)] = '\n'
	if t.Session == nil {
		err = t.Initialize()
		if err != nil {
			err = errors.Wrapf(err, "failed to send telnet message: %s", s)
			return
		}
	}
	_, err = t.Session.Write(buf)
	if err != nil {
		err = errors.Wrapf(err, "failed to write telnet message: %s", s)
		return
	}
	return
}

func (t *Telnet) alphanumeric(data string) (sData string) {
	sData = data
	re := regexp.MustCompile("[^a-zA-Z0-9_]+")
	sData = re.ReplaceAllString(sData, "")
	return
}

func (t *Telnet) convertLinks(message string) (messageFixed string) {
	prefix := t.ItemURL
	messageFixed = message
	if strings.Count(message, "") > 1 {
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
		messageFixed = sets[0] + itemlink + sets[2]
		if strings.Count(message, "") > 1 {
			messageFixed = t.convertLinks(messageFixed)
		}
	}
	return
}
