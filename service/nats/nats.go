package nats

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/nats-io/go-nats"
	"github.com/pkg/errors"
	"github.com/xackery/talkeq/model"
	"github.com/xackery/talkeq/service"
)

// Nats represents the nats service
type Nats struct {
	Username    string
	Password    string
	Host        string
	Port        string
	ItemURL     string
	Session     *nats.Conn
	Log         *log.Logger
	Switchboard *service.Switchboard
}

// Name returns the name of the service
func (n *Nats) Name() string {
	return "nats"
}

// Initialize starts a new nats session
func (n *Nats) Initialize() (err error) {
	if n.Log == nil {
		err = errors.New("log not initialized")
		return
	}

	if len(n.Host) > 0 {
		n.Session, err = nats.Connect(fmt.Sprintf("nats://%s:%s", n.Host, n.Port))
		if err != nil {
			n.Session, err = nats.Connect(nats.DefaultURL)
			if err != nil {
				err = errors.Wrap(err, "failed to connect to nats")
				return
			}
		}
	} else {
		n.Session, err = nats.Connect(nats.DefaultURL)
		if err != nil {
			err = errors.Wrap(err, "failed to connect to nats")
			return
		}
	}

	//n.Session.Subscribe("world.daily_gain.out", n.onDailyGainEvent)
	n.Session.Subscribe("world.channel_message.out", n.onChannelMessageEvent)
	//n.Session.Subscribe("global.admin_message.out", n.onAdminMessageEvent)
	return
}

// Close will close the nats session.
func (n *Nats) Close() (err error) {
	n.Session.Close()
	n.Session = nil
	return
}

// SendChannelMessage handles message requests
func (n *Nats) SendChannelMessage(message *model.ChannelMessage) (err error) {

	if n.Session == nil {
		err = n.Initialize()
		if err != nil {
			err = errors.Wrap(err, "failed to write message")
			return
		}
	}

	message.Message = fmt.Sprintf("%s says from %s, '%s'", message.From, message.Creator, message.Message)
	msg, err := proto.Marshal(message)
	if err != nil {
		err = errors.Wrap(err, "failed to marshal message")
		return
	}

	err = n.Session.Publish("world.channel_message.in", msg)
	if err != nil {
		err = errors.Wrap(err, "failed to publish message")
		return
	}

	return
}

// SendCommandMessage handles message requests
func (n *Nats) SendCommandMessage(message *model.CommandMessage) (err error) {
	if n.Session == nil {
		err = n.Initialize()
		if err != nil {
			err = errors.Wrap(err, "failed to write message")
			return
		}
	}

	msg, err := proto.Marshal(message)
	if err != nil {
		err = errors.Wrap(err, "failed to marshal command")
		return
	}
	err = n.Session.Publish("world.command_message.in", msg)
	if err != nil {
		err = errors.Wrap(err, "failed to publish message")
		return
	}

	return
}

func (n *Nats) onChannelMessageEvent(nm *nats.Msg) {
	message := &model.ChannelMessage{}
	err := proto.Unmarshal(nm.Data, message)
	if err != nil {
		err = errors.Wrap(err, "failed to unmarshal channel message")
		return
	}
	message.Creator = n.Name()
	message.From = strings.Replace(message.From, "_", " ", -1)
	if strings.Contains(message.From, " ") {
		message.From = fmt.Sprintf("%s [%s]", message.From[:strings.Index(message.From, " ")], message.From[strings.Index(message.From, " ")+1:])
	}
	message.From = n.alphanumeric(message.From) //purify name to be alphanumeric

	if strings.Contains(message.Message, "Summoning you to") { //GM messages are relaying to discord!
		return
	}
	message.Message = n.convertLinks(message.Message)

	n.Log.Printf("message: [%d] %s: %s", message.Number, message.From, message.Message)
	services := n.Switchboard.FindPatch(message.Number, n)
	for _, service := range services {
		err = service.SendChannelMessage(message)
		if err != nil {
			n.Log.Printf("-> %s failed: %s\n", service.Name(), err.Error())
			continue
		}
	}

}

func (n *Nats) sanitize(data string) (sData string) {
	sData = data
	sData = strings.Replace(sData, `%`, "&PCT;", -1)
	re := regexp.MustCompile("[^\x00-\x7F]+")
	sData = re.ReplaceAllString(sData, "")
	return
}

func (n *Nats) alphanumeric(data string) (sData string) {
	sData = data
	re := regexp.MustCompile("[^a-zA-Z0-9_]+")
	sData = re.ReplaceAllString(sData, "")
	return
}

func (n *Nats) convertLinks(message string) (messageFixed string) {
	prefix := n.ItemURL
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
			messageFixed = n.convertLinks(messageFixed)
		}
	}
	return
}
