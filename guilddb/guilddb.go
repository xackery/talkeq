package guilddb

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/xackery/log"
	"github.com/xackery/talkeq/config"
)

var (
	isStarted          bool
	guilds             map[int]string
	mu                 sync.RWMutex
	guildsDatabasePath string
)

// New creates a new guild database
func New(config *config.Config) error {
	if isStarted {
		return fmt.Errorf("already started")
	}
	guildsDatabasePath = config.GuildsDatabasePath

	log := log.New()
	log.Debug().Msgf("initializing guild db")
	_, err := os.Stat(guildsDatabasePath)
	if os.IsNotExist(err) {
		err = ioutil.WriteFile(guildsDatabasePath, []byte(`# guildid:channelid #comment`), 0644)
		if err != nil {
			return fmt.Errorf("guilds database create %w", err)
		}
	}

	err = reload()
	if err != nil {
		return fmt.Errorf("reload: %w", err)
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("newWatcher: %w", err)
	}

	err = watcher.Add(config.GuildsDatabasePath)
	if err != nil {
		return fmt.Errorf("watcherAdd: %w", err)
	}

	go loop(watcher)
	return nil
}

func loop(watcher *fsnotify.Watcher) {
	if isStarted {
		return
	}
	mu.Lock()
	isStarted = true
	mu.Unlock()
	log := log.New()

	defer watcher.Close()
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				log.Warn().Msg("guild database failed to read file")
				return
			}
			if event.Op&fsnotify.Write != fsnotify.Write {
				continue
			}
			log.Debug().Msg("guilds database modified, reloading")
			err := reload()
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

func reload() error {
	mu.Lock()
	defer mu.Unlock()
	data, err := ioutil.ReadFile(guildsDatabasePath)
	if err != nil {
		return fmt.Errorf("readFile: %w", err)
	}
	log := log.New()

	ng := make(map[int]string)
	lines := strings.Split(string(data), "\n")
	for lineNumber, line := range lines {
		lineNumber++
		p := strings.Index(line, ":")
		if p < 1 {
			log.Warn().Int("line number", lineNumber).Msgf("%s no : exists", guildsDatabasePath)
			continue
		}
		sid := line[0:p]
		if len(sid) < 1 {
			log.Warn().Int("line number", lineNumber).Msgf("%s guildid too short", guildsDatabasePath)
			continue
		}
		iid, err := strconv.Atoi(sid)
		if err != nil {
			log.Warn().Int("line number", lineNumber).Msgf("%s guildid not valid int", guildsDatabasePath)
			continue
		}
		id := int(iid)
		name := line[p+1:]
		if len(name) < 3 {
			log.Warn().Int("line number", lineNumber).Msgf("%s guildname too short", guildsDatabasePath)
			continue
		}
		p = strings.Index(name, "#")
		if p > 0 {
			name = name[0:p]
		}
		name = strings.TrimSpace(name)
		_, ok := ng[id]
		if ok {
			log.Warn().Int("line number", lineNumber).Int("guild id", id).Msgf("%s duplicate entry", guildsDatabasePath)
		}
		ng[id] = name
	}

	guilds = ng
	return nil
}

// Set updates or adds an entry for a specified guild id
func Set(guildID int, guildName string) {
	mu.Lock()
	defer mu.Unlock()
	guilds[guildID] = guildName
}

// ChannelID returns the discord ChannelID of a guild based on their ID
func ChannelID(guildID int) string {
	mu.RLock()
	defer mu.RUnlock()
	return guilds[guildID]
}

// GuildId returns the EQ guildID of a guild based on a provided discord channelID, returns 0 if no results
func GuildID(channelID string) int {
	mu.RLock()
	defer mu.RUnlock()
	for guildID, gChannelID := range guilds {
		if channelID == gChannelID {
			return guildID
		}
	}
	return 0
}
