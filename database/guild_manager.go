package database

import (
	"context"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/xackery/talkeq/config"
)

// GuildManager manages the guilds registered
type GuildManager struct {
	ctx                context.Context
	guilds             map[int]string
	mutex              sync.RWMutex
	guildsDatabasePath string
}

// NewGuildManager instantiates a new guilds database manager
func NewGuildManager(ctx context.Context, config *config.Config) (*GuildManager, error) {
	u := new(GuildManager)
	u.guildsDatabasePath = config.GuildsDatabasePath
	u.guilds = make(map[int]string)

	_, err := os.Stat(u.guildsDatabasePath)
	if os.IsNotExist(err) {
		err = ioutil.WriteFile(u.guildsDatabasePath, []byte(`# guildid:channelid #comment`), 0644)
		if err != nil {
			return nil, errors.Wrapf(err, "guilds database create %s", u.guildsDatabasePath)
		}
	}

	err = u.reloadDatabase()
	if err != nil {
		return nil, errors.Wrap(err, "reloadDatabase")
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, errors.Wrap(err, "newwatcher")
	}

	err = watcher.Add(config.GuildsDatabasePath)
	if err != nil {
		return nil, errors.Wrap(err, "watcheradd")
	}

	go u.loop(ctx, watcher)
	return u, nil
}

func (u *GuildManager) loop(ctx context.Context, watcher *fsnotify.Watcher) {
	defer watcher.Close()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				log.Warn().Msg("guild database failed to read file")
				return
			}
			if event.Op&fsnotify.Write != fsnotify.Write {
				continue
			}
			log.Debug().Msg("guilds database modified, reloading")
			err := u.reloadDatabase()
			if err != nil {
				log.Warn().Err(err).Msg("failed to reload guilds database")
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Warn().Err(err).Msg("guild database failed to read file")
		}
	}
}

func (u *GuildManager) reloadDatabase() error {
	data, err := ioutil.ReadFile(u.guildsDatabasePath)
	if err != nil {
		return errors.Wrap(err, "reloadDatabase")
	}

	nu := make(map[int]string)
	lines := strings.Split(string(data), "\n")
	for lineNumber, line := range lines {
		lineNumber++
		p := strings.Index(line, ":")
		if p < 1 {
			log.Warn().Int("line number", lineNumber).Msgf("%s no : exists", u.guildsDatabasePath)
			continue
		}
		sid := line[0:p]
		if len(sid) < 1 {
			log.Warn().Int("line number", lineNumber).Msgf("%s guildid too short", u.guildsDatabasePath)
			continue
		}
		iid, err := strconv.Atoi(sid)
		if err != nil {
			log.Warn().Int("line number", lineNumber).Msgf("%s guildid not valid int", u.guildsDatabasePath)
			continue
		}
		id := int(iid)
		name := line[p+1:]
		if len(name) < 3 {
			log.Warn().Int("line number", lineNumber).Msgf("%s guildname too short", u.guildsDatabasePath)
			continue
		}
		p = strings.Index(name, "#")
		if p > 0 {
			name = name[0:p]
		}
		name = strings.TrimSpace(name)
		_, ok := nu[id]
		if ok {
			log.Warn().Int("line number", lineNumber).Int("guild id", id).Msgf("%s duplicate entry", u.guildsDatabasePath)
		}
		nu[id] = name
	}

	u.mutex.Lock()
	u.guilds = nu
	u.mutex.Unlock()
	return nil
}

// Set updates or adds an entry for a specified guild id
func (u *GuildManager) Set(guildID int, guildName string) {
	u.mutex.Lock()
	u.guilds[guildID] = guildName
	u.mutex.Unlock()
}

// ChannelID returns the discord ChannelID of a guild based on their ID
func (u *GuildManager) ChannelID(guildID int) string {
	u.mutex.RLock()
	name := u.guilds[guildID]
	u.mutex.RUnlock()
	return name
}
