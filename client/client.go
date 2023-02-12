package client

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/xackery/talkeq/api"
	"github.com/xackery/talkeq/config"
	"github.com/xackery/talkeq/discord"
	"github.com/xackery/talkeq/eqlog"
	"github.com/xackery/talkeq/guilddb"
	"github.com/xackery/talkeq/peqeditorsql"
	"github.com/xackery/talkeq/request"
	"github.com/xackery/talkeq/sqlreport"
	"github.com/xackery/talkeq/telnet"
	"github.com/xackery/talkeq/tlog"
	"github.com/xackery/talkeq/userdb"
)

// Client wraps all talking endpoints
type Client struct {
	ctx          context.Context
	cancel       context.CancelFunc
	config       *config.Config
	discord      *discord.Discord
	telnet       *telnet.Telnet
	eqlog        *eqlog.EQLog
	sqlreport    *sqlreport.SQLReport
	peqeditorsql *peqeditorsql.PEQEditorSQL
	api          *api.API
}

// New creates a new client
func New(ctx context.Context) (*Client, error) {
	var err error
	ctx, cancel := context.WithCancel(ctx)
	c := Client{
		ctx:    ctx,
		cancel: cancel,
	}
	tlog.Debugf("[talkeq] initializing talkeq client")
	c.config, err = config.NewConfig(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "config")
	}

	tlog.Debugf("[talkeq] initializing databases")
	err = userdb.New(c.config)
	if err != nil {
		return nil, fmt.Errorf("userdb.New: %w", err)
	}

	err = guilddb.New(c.config)
	if err != nil {
		return nil, fmt.Errorf("guilddb.New: %w", err)
	}

	tlog.Debugf("[talkeq] initializing 3rd party connections")
	c.discord, err = discord.New(ctx, c.config.Discord)
	if err != nil {
		return nil, errors.Wrap(err, "discord")
	}

	err = c.discord.Subscribe(ctx, c.onMessage)
	if err != nil {
		return nil, errors.Wrap(err, "discord subscribe")
	}

	c.telnet, err = telnet.New(ctx, c.config.Telnet)
	if err != nil {
		return nil, errors.Wrap(err, "telnet")
	}

	c.sqlreport, err = sqlreport.New(ctx, c.config.SQLReport, c.discord)
	if err != nil {
		return nil, errors.Wrap(err, "sqlreport")
	}

	err = c.telnet.Subscribe(ctx, c.onMessage)
	if err != nil {
		return nil, errors.Wrap(err, "telnet subscribe")
	}

	c.eqlog, err = eqlog.New(ctx, c.config.EQLog)
	if err != nil {
		return nil, errors.Wrap(err, "eqlog")
	}

	err = c.eqlog.Subscribe(ctx, c.onMessage)
	if err != nil {
		return nil, errors.Wrap(err, "eqlog subscribe")
	}

	c.peqeditorsql, err = peqeditorsql.New(ctx, c.config.PEQEditor.SQL)
	if err != nil {
		return nil, errors.Wrap(err, "peqeditorsql")
	}

	err = c.peqeditorsql.Subscribe(ctx, c.onMessage)
	if err != nil {
		return nil, errors.Wrap(err, "peqeditorsql subscribe")
	}

	tlog.Debugf("[talkeq] initializing API")
	c.api, err = api.New(ctx, c.config.API, c.discord)
	if err != nil {
		return nil, errors.Wrap(err, "api subscribe")
	}

	err = c.api.Subscribe(ctx, c.onMessage)
	if err != nil {
		return nil, errors.Wrap(err, "api subscribe")
	}

	return &c, nil
}

// Connect attempts to connect to all enabled endpoints
func (c *Client) Connect(ctx context.Context) error {
	tlog.Debugf("[talkeq] connecting")

	err := c.discord.Connect(ctx)
	if err != nil {
		if !c.config.IsKeepAliveEnabled {
			return errors.Wrap(err, "discord connect")
		}
		tlog.Warnf("[discord] connect failed: %s", err)
	}

	err = c.telnet.Connect(ctx)
	if err != nil {
		if !c.config.IsKeepAliveEnabled {
			return errors.Wrap(err, "telnet connect")
		}
		tlog.Warnf("[telnet] connect failed: %s", err)
	}

	err = c.sqlreport.Connect(ctx)
	if err != nil {
		if !c.config.IsKeepAliveEnabled {
			return errors.Wrap(err, "sqlreport connect")
		}
		tlog.Warnf("[sqlreport] connect failed: %s", err)
	}

	err = c.eqlog.Connect(ctx)
	if err != nil {
		if !c.config.IsKeepAliveEnabled {
			return errors.Wrap(err, "eqlog connect")
		}
		tlog.Warnf("[eqlog] connect failed: %s", err)
	}

	err = c.peqeditorsql.Connect(ctx)
	if err != nil {
		if !c.config.IsKeepAliveEnabled {
			return errors.Wrap(err, "peqeditorsql connect")
		}
		tlog.Warnf("[peqeditorsql] connect failed: %s", err)
	}

	err = c.api.Connect(ctx)
	if err != nil {
		if !c.config.IsKeepAliveEnabled {
			return errors.Wrap(err, "api connect")
		}
		tlog.Warnf("[api] connect failed: %s", err)
	}

	go c.loop(ctx)
	return nil
}

func (c *Client) loop(ctx context.Context) {
	var err error
	go func() {
		var err error
		var online int
		for {
			select {
			case <-ctx.Done():
				tlog.Debugf("[talkeq] status loop exit, context done")
				return
			default:
			}
			if c.config.Telnet.IsEnabled && c.config.Discord.IsEnabled {
				online, err = c.telnet.Who(ctx)
				if err != nil {
					tlog.Warnf("[telnet] who failed: %s", err)
				}
				err = c.discord.StatusUpdate(ctx, online, "")
				if err != nil {
					tlog.Warnf("[discord] status update failed: %s", err)
				}
			}

			time.Sleep(60 * time.Second)
		}
	}()
	if !c.config.IsKeepAliveEnabled {
		tlog.Debugf("[talkeq] keep_alive disabled in config, exiting client loop")
		return
	}
	for {
		select {
		case <-ctx.Done():
			tlog.Debugf("[talkeq] client loop exit, context done")
			return
		default:
		}
		time.Sleep(c.config.KeepAliveRetryDuration())
		if c.config.Discord.IsEnabled && !c.discord.IsConnected() {
			tlog.Infof("[discord] attempting to reconnect")
			err = c.discord.Connect(ctx)
			if err != nil {
				tlog.Warnf("[discord] reconnect failed: %s", err)
			}
		}
		if c.config.Telnet.IsEnabled && !c.telnet.IsConnected() {
			tlog.Infof("[telnet] attempting to reconnect")
			err = c.telnet.Connect(ctx)
			if err != nil {
				tlog.Warnf("[telnet] reconnect failed: %s", err)
			}
		}
		if c.config.SQLReport.IsEnabled && !c.sqlreport.IsConnected() {
			tlog.Infof("[sqlreport] attempting to reconnect")
			err = c.sqlreport.Connect(ctx)
			if err != nil {
				tlog.Warnf("[sqlreport] connect failed: %s", err)
			}
		}
	}
}

func (c *Client) onMessage(rawReq interface{}) error {
	var err error

	switch req := rawReq.(type) {
	case request.APICommand:
		err = c.api.Command(req)
	case request.DiscordSend:
		err = c.discord.Send(req)
	case request.TelnetSend:
		err = c.telnet.Send(req)
	default:
		return fmt.Errorf("unknown request type")
	}
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}
	return nil
}

// Disconnect attempts to gracefully disconnect all enabled endpoints
func (c *Client) Disconnect(ctx context.Context) error {
	err := c.discord.Disconnect(ctx)
	if err != nil {
		return errors.Wrap(err, "discord")
	}
	c.cancel()
	return nil
}
