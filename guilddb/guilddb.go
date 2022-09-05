package guilddb

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/xackery/talkeq/config"
	"github.com/xackery/talkeq/tlog"
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

	tlog.Debugf("[guilddb] initializing")
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

	defer watcher.Close()
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				tlog.Warn("[guilddb] failed to read file")
				return
			}
			if event.Op&fsnotify.Write != fsnotify.Write {
				continue
			}
			tlog.Debugf("[guilddb] modified, reloading")
			err := reload()
			if err != nil {
				tlog.Warnf("[guilddb] failed to reload: %s", err)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			tlog.Warnf("[guilddb] failed to read file: %s", err)
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

	ng := make(map[int]string)
	lines := strings.Split(string(data), "\n")
	for lineNumber, line := range lines {
		lineNumber++

		line = strings.TrimSpace(line)
		if len(line) < 1 {
			//tlog.Debugf("[guilddb] line %d skipped, empty", lineNumber)
			continue
		}
		if line[0] == '#' {
			continue
		}
		p := strings.Index(line, ":")
		if p < 1 {
			tlog.Debugf("[guilddb] line %d skipped, no : found", lineNumber)
			continue
		}
		sid := line[0:p]
		if len(sid) < 1 {
			tlog.Warnf("[guilddb] line %d failed, guildid too short", lineNumber)
			continue
		}
		iid, err := strconv.Atoi(sid)
		if err != nil {
			tlog.Warnf("[guilddb] line %d failed, guildid not a valid integer", lineNumber)
			continue
		}
		id := int(iid)
		name := line[p+1:]
		if len(name) < 3 {
			tlog.Warnf("[guilddb] line %d failed, guildname too short", lineNumber)
			continue
		}
		p = strings.Index(name, "#")
		if p > 0 {
			name = name[0:p]
		}
		name = strings.TrimSpace(name)
		_, ok := ng[id]
		if ok {
			tlog.Debugf("[guilddb] line %d skipped, guildID %d is a duplicate entry", lineNumber, id)
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
