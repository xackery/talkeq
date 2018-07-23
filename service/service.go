package service

import (
	"github.com/xackery/talkeq/model"
)

// Service represents any service
type Service interface {
	Name() string
	Initialize() (err error)
	WriteMessage(message *model.Message) (err error)
	Close() (err error)
}
