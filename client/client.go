package client

import (
	"context"
	"os"
	"runtime"
	"time"

	"github.com/xackery/talkeq/channel"

	"github.com/xackery/talkeq/config"
	"github.com/xackery/talkeq/discord"
	"github.com/xackery/talkeq/eqlog"
	"github.com/xackery/talkeq/telnet"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Client wraps all talking endpoints
type Client struct {
	ctx     context.Context
	cancel  context.CancelFunc
	config  *config.Config
	discord *discord.Discord
	telnet  *telnet.Telnet
	eqlog   *eqlog.EQLog
}

// New creates a new client
func New(ctx context.Context) (*Client, error) {
	var err error
	ctx, cancel := context.WithCancel(ctx)
	c := Client{
		ctx:    ctx,
		cancel: cancel,
	}
	if runtime.GOOS != "windows" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}
	c.config, err = config.NewConfig(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "config")
	}

	if c.config.IsKeepAliveEnabled && c.config.KeepAliveRetry.Seconds() < 2 {
		c.config.KeepAliveRetry.Duration = 10 * time.Second
		//return nil, fmt.Errorf("keep_alive_retry must be greater than 2s")
	}

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
	return &c, nil
}

// Connect attempts to connect to all enabled endpoints
func (c *Client) Connect(ctx context.Context) error {
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

	err = c.eqlog.Connect(ctx)
	if err != nil {
		if !c.config.IsKeepAliveEnabled {
			return errors.Wrap(err, "eqlog connect")
		}
		log.Warn().Err(err).Msg("eqlog connect")
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
		time.Sleep(c.config.KeepAliveRetry.Duration)
		if c.config.Discord.IsEnabled && !c.discord.IsConnected() {
			log.Info().Msg("attempting to reconnect to discord")
			err = c.discord.Connect(ctx)
			if err != nil {
				log.Warn().Err(err).Msg("discord connect")
			}
		}
	}
}

func (c *Client) onMessage(source string, author string, channelID int, message string) {
	var err error
	endpoints := "none"
	switch source {
	case "telnet":
		if !c.config.Discord.IsEnabled {
			log.Info().Msgf("[%s->none] %s %s: %s", source, author, channel.ToString(channelID), message)
			return
		}
		err = c.discord.Send(context.Background(), source, author, channelID, message)
		if err != nil {
			log.Warn().Err(err).Msg("discord send")
		} else {
			if endpoints == "none" {
				endpoints = "discord"
			} else {
				endpoints += ",discord"
			}
		}
		log.Info().Msgf("[%s->%s] %s %s: %s", source, endpoints, author, channel.ToString(channelID), message)
	case "discord":
		if !c.config.Telnet.IsEnabled {
			log.Info().Msgf("[%s->none] %s %s: %s", source, author, channel.ToString(channelID), message)
			return
		}
		err = c.telnet.Send(context.Background(), source, author, channelID, message)
		if err != nil {
			log.Warn().Err(err).Msg("telnet send")
		} else {
			if endpoints == "none" {
				endpoints = "telnet"
			} else {
				endpoints += ",telnet"
			}
		}
		log.Info().Msgf("[%s->%s] %s %s: %s", source, endpoints, author, channel.ToString(channelID), message)
	case "eqlog":
		if !c.config.Discord.IsEnabled {
			log.Info().Msgf("[%s->none] %s %s: %s", source, author, channel.ToString(channelID), message)
			return
		}
		err = c.discord.Send(context.Background(), source, author, channelID, message)
		if err != nil {
			log.Warn().Err(err).Msg("discord send")
		} else {
			if endpoints == "none" {
				endpoints = "discord"
			} else {
				endpoints += ",discord"
			}
		}
		log.Info().Msgf("[%s->%s] %s %s: %s", source, endpoints, author, channel.ToString(channelID), message)
	default:
		log.Warn().Str("source", source).Str("author", author).Int("channelID", channelID).Str("message", message).Msg("unknown source")
	}
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
