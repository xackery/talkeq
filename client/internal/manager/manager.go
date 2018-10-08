package manager

import (
	"context"
	"time"

	"github.com/xackery/talkeq/model"
)

var (
	queryTimeout = 3 * time.Second
)

// Manager implements the talkEQ system
type Manager struct {
	ctx             context.Context
	queryChan       chan *model.QueryRequest
	config          *model.ConfigEndpoint
	clientQueryChan chan *model.QueryRequest
}

// New creates a new client
func New(ctx context.Context, config *model.ConfigEndpoint, clientQueryChan chan *model.QueryRequest) (m *Manager, err error) {
	m = &Manager{
		ctx:             ctx,
		queryChan:       make(chan *model.QueryRequest),
		config:          config,
		clientQueryChan: clientQueryChan,
	}
	go m.pump()
	return
}
