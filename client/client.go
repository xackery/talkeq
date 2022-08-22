package client

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/xackery/log"
	"github.com/xackery/talkeq/api"
	"github.com/xackery/talkeq/config"
	"github.com/xackery/talkeq/discord"
	"github.com/xackery/talkeq/eqlog"
	"github.com/xackery/talkeq/guilddb"
	"github.com/xackery/talkeq/peqeditorsql"
	"github.com/xackery/talkeq/request"
	"github.com/xackery/talkeq/sqlreport"
	"github.com/xackery/talkeq/telnet"
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
	log := log.New()
	log.Debug().Msgf("initializing talkeq client")
	c.config, err = config.NewConfig(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "config")
	}

	log.Debug().Msgf("initializing databases")
	err = userdb.New(c.config)
	if err != nil {
		return nil, fmt.Errorf("userdb.New: %w", err)
	}

	err = guilddb.New(c.config)
	if err != nil {
		return nil, fmt.Errorf("guilddb.New: %w", err)
	}

	log.Debug().Msgf("initializing 3rd party connections")
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

	log.Debug().Msgf("initializing API")
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
	log := log.New()
	log.Debug().Msgf("talkeq connecting")

	err := c.discord.Connect(ctx)
	if err != nil {
		if !c.config.IsKeepAliveEnabled {
			return errors.Wrap(err, "discord connect")
		}
		log.Warn().Err(err).Msg("discord connect")
	}

	err = c.telnet.Connect(ctx)
	if err != nil {
		if !c.config.IsKeepAliveEnabled {
			return errors.Wrap(err, "telnet connect")
		}
		log.Warn().Err(err).Msg("telnet connect")
	}

	err = c.sqlreport.Connect(ctx)
	if err != nil {
		if !c.config.IsKeepAliveEnabled {
			return errors.Wrap(err, "sqlreport connect")
		}
		log.Warn().Err(err).Msg("sqlreport connect")
	}

	err = c.eqlog.Connect(ctx)
	if err != nil {
		if !c.config.IsKeepAliveEnabled {
			return errors.Wrap(err, "eqlog connect")
		}
		log.Warn().Err(err).Msg("eqlog connect")
	}

	err = c.peqeditorsql.Connect(ctx)
	if err != nil {
		if !c.config.IsKeepAliveEnabled {
			return errors.Wrap(err, "peqeditorsql connect")
		}
		log.Warn().Err(err).Msg("peqeditorsql connect")
	}

	err = c.api.Connect(ctx)
	if err != nil {
		if !c.config.IsKeepAliveEnabled {
			return errors.Wrap(err, "api connect")
		}
		log.Warn().Err(err).Msg("api connect")
	}

	go c.loop(ctx)
	return nil
}

func (c *Client) loop(ctx context.Context) {
	log := log.New()
	var err error
	go func() {
		var err error
		var online int
		for {
			select {
			case <-ctx.Done():
				log.Debug().Msg("status loop exit, context done")
				return
			default:
			}
			if c.config.Telnet.IsEnabled && c.config.Discord.IsEnabled {
				online, err = c.telnet.Who(ctx)
				if err != nil {
					log.Warn().Err(err).Msg("telnet who")
				}
				err = c.discord.StatusUpdate(ctx, online, "")
				if err != nil {
					log.Warn().Err(err).Msg("discord status update")
				}
			}

			time.Sleep(60 * time.Second)
		}
	}()
	if !c.config.IsKeepAliveEnabled {
		log.Debug().Msg("keep_alive disabled in config, exiting client loop")
		return
	}
	for {
		select {
		case <-ctx.Done():
			log.Debug().Msg("client loop exit, context done")
			return
		default:
		}
		time.Sleep(c.config.KeepAliveRetryDuration())
		if c.config.Discord.IsEnabled && !c.discord.IsConnected() {
			log.Info().Msg("attempting to reconnect to discord")
			err = c.discord.Connect(ctx)
			if err != nil {
				log.Warn().Err(err).Msg("discord connect")
			}
		}
		if c.config.Telnet.IsEnabled && !c.telnet.IsConnected() {
			log.Info().Msg("attempting to reconnect to telnet")
			err = c.telnet.Connect(ctx)
			if err != nil {
				log.Warn().Err(err).Msg("telnet connect")
			}
		}
		if c.config.SQLReport.IsEnabled && !c.sqlreport.IsConnected() {
			log.Info().Msg("attempting to reconnect to sqlreport")
			err = c.sqlreport.Connect(ctx)
			if err != nil {
				log.Warn().Err(err).Msg("sqlreport connect")
			}
		}
	}
}

func (c *Client) onMessage(rawReq interface{}) error {
	var err error

	switch rawReq.(type) {
	case request.APICommand:
		err = c.api.Command(rawReq.(request.APICommand))
	case request.DiscordSend:
		err = c.discord.Send(rawReq.(request.DiscordSend))
	case request.TelnetSend:
		err = c.telnet.Send(rawReq.(request.TelnetSend))
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
