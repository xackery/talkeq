package client

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/xackery/talkeq/client/internal/endpoint"
	"github.com/xackery/talkeq/client/internal/endpoint/discord"
	"github.com/xackery/talkeq/client/internal/endpoint/nats"
	"github.com/xackery/talkeq/model"
)

// Start begins talkeq
func (c *Client) Start(ctx context.Context) (err error) {
	_, err = c.runQuery(ctx, "Start", nil)
	if err != nil {
		err = errors.Wrap(err, "failed to start")
		return
	}
	return
}

func (c *Client) onStart(ctx context.Context) (err error) {
	var end endpoint.Endpointer
	if c.config.Discord.Enabled {
		end, err = discord.New(ctx, c.manager)
		if err != nil {
			err = errors.Wrap(err, "failed to initialize discord")
			return
		}
		c.endpoints["discord"] = end
	}
	if c.config.NATS.Enabled {
		end, err = nats.New(ctx, c.manager)
		if err != nil {
			err = errors.Wrap(err, "failed to initialize nats")
			return
		}
		c.endpoints["nats"] = end
	}
	if c.config.Telnet.Enabled {
		end, err = nats.New(ctx, c.manager)
		if err != nil {
			err = errors.Wrap(err, "failed to initialize telnet")
			return
		}
		c.endpoints["telnet"] = end
	}
	if c.config.EQLog.Enabled {
		end, err = nats.New(ctx, c.manager)
		if err != nil {
			err = errors.Wrap(err, "failed to initialize eqlog")
			return
		}
		c.endpoints["eqlog"] = end
	}
	if len(c.endpoints) == 0 {
		err = fmt.Errorf("all endpoints are disabled, please enable one")
		return
	}

	for name, end := range c.endpoints {
		err = end.ConfigUpdate(ctx, c.config)
		if err != nil {
			err = errors.Wrapf(err, "failed to set config for %s", name)
			return
		}
		err = end.Connect(ctx)
		if err != nil {
			err = errors.Wrapf(err, "failed to connect %s", name)
			return
		}
	}

	logger := model.NewLogger(ctx)
	logger.Info().Msgf("started %d services", len(c.endpoints))
	return
}
