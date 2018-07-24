package eqlog

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/hpcloud/tail"
	"github.com/pkg/errors"
	"github.com/xackery/talkeq/model"
	"github.com/xackery/talkeq/service"
)

// EQLog represents the eqlog.txt service
type EQLog struct {
	Username    string
	Password    string
	Log         *log.Logger
	File        string
	newTelnet   bool
	ItemURL     string
	IP          string
	Port        string
	Switchboard *service.Switchboard
	isPolling   bool
}

// Name returns the name of the service
func (e *EQLog) Name() string {
	return "eqlog"
}

// Initialize starts a new telnet session
func (e *EQLog) Initialize() (err error) {
	if e.Log == nil {
		err = errors.New("log not initialized")
		return
	}
	if e.File == "" {
		err = errors.New("file to watch not specified")
		return
	}

	go e.pollMessages()
	return
}

// Close will close the discord session.
func (e *EQLog) Close() (err error) {
	e.isPolling = false
	return
}

// SendChannelMessage handles message requests
func (e *EQLog) SendChannelMessage(message *model.ChannelMessage) (err error) {
	err = errors.New("eqlog does not support writing messages")
	return
}

// SendCommandMessage handles message requests
func (e *EQLog) SendCommandMessage(message *model.CommandMessage) (err error) {
	err = errors.New("eqlog does not support writing messages")
	return
}

func (e *EQLog) pollMessages() {
	message := &model.ChannelMessage{
		Creator: e.Name(),
	}
	var err error

	t, err := tail.TailFile(e.File, tail.Config{Follow: true})
	for line := range t.Lines {
		message.Message = ""
		message.Number = 260
		message.From = ""
		message.Message = line.Text
		if !e.isPolling {
			return
		}

		if len(message.Message) < 3 { //ignore small messages
			continue
		}

		//todo: Add support for new message types
		if !strings.Contains(message.Message, "says out of character,") { //ignore non-ooc
			continue
		}

		message.From = message.Message[0:strings.Index(message.Message, " says out of character,")]
		if strings.Contains(message.From, "]") {
			message.From = message.From[strings.Index(message.From, "]")+1 : len(message.From)]
		}

		message.From = e.alphanumeric(message.From)

		message.Message = message.Message[strings.Index(message.Message, "says out of character, '")+24 : len(message.Message)-1]
		message.From = strings.Replace(message.From, "_", " ", -1)
		message.Message = e.convertLinks(message.Message) //This may work in log poller

		//Todo: Send message to writemessage
		e.Log.Printf("message: [%d] %s: %s", message.Number, message.From, message.Message)
		services := e.Switchboard.FindPatch(message.Number, e)
		for _, service := range services {
			err = service.SendChannelMessage(message)
			if err != nil {
				e.Log.Printf("-> %s failed: %s\n", service.Name(), err.Error())
				continue
			}
		}
	}
}

func (e *EQLog) alphanumeric(data string) (sData string) {
	sData = data
	re := regexp.MustCompile("[^a-zA-Z0-9_]+")
	sData = re.ReplaceAllString(sData, "")
	return
}

func (e *EQLog) convertLinks(message string) (messageFixed string) {
	prefix := e.ItemURL
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
			messageFixed = e.convertLinks(messageFixed)
		}
	}
	return
}
