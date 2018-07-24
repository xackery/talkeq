package service

import (
	"github.com/xackery/talkeq/model"
)

// Service represents any service
type Service interface {
	Name() string
	Initialize() (err error)
	SendChannelMessage(message *model.ChannelMessage) (err error)
	SendCommandMessage(message *model.CommandMessage) (err error)
	Close() (err error)
}
