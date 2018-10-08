package client

import (
	"context"
	"os"
	"runtime"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/xackery/talkeq/client/internal/endpoint"
	"github.com/xackery/talkeq/client/internal/manager"
	"github.com/xackery/talkeq/model"
)

var (
	queryTimeout = 3 * time.Second
)

// Client implements the talkEQ system
type Client struct {
	ctx       context.Context
	queryChan chan *model.QueryRequest
	endpoints map[string]endpoint.Endpointer
	manager   *manager.Manager
	config    *model.ConfigEndpoint
}

// New creates a new client
func New(ctx context.Context) (c *Client, err error) {
	queryChan := make(chan *model.QueryRequest)
	config := model.NewConfigEndpoint()
	manager, err := manager.New(ctx, config, queryChan)
	if err != nil {
		err = errors.Wrap(err, "failed to create manager")
		return
	}
	c = &Client{
		ctx:       ctx,
		queryChan: queryChan,
		endpoints: make(map[string]endpoint.Endpointer),
		manager:   manager,
		config:    config,
	}
	if runtime.GOOS != "windows" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
	go c.pump()
	return
}
