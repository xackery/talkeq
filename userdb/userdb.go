package userdb

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/jbsmith7741/toml"
	"github.com/xackery/talkeq/config"
	"github.com/xackery/talkeq/tlog"
)

var (
	isStarted         bool
	mu                sync.RWMutex
	users             map[string]UserEntry
	usersDatabasePath string
)

// UserEntry represents a record in the database
type UserEntry struct {
	CharacterName string
	DiscordID     string
}

// New initializes and creates the user database
func New(config *config.Config) error {
	if isStarted {
		return fmt.Errorf("already started")
	}
	usersDatabasePath = config.UsersDatabasePath

	tlog.Debugf("[userdb] initializing user db")
	ext := filepath.Ext(usersDatabasePath)
	_, err := os.Stat(usersDatabasePath)
	if os.IsNotExist(err) {
		tlog.Debugf("[userdb] not found, creating a new one")
		f, err := os.Create(usersDatabasePath)
		if err != nil {
			return fmt.Errorf("create user database: %w", err)
		}
		defer f.Close()
		if ext == ".toml" {
			enc := toml.NewEncoder(f)
			mu.Lock()
			err = enc.Encode(users)
			mu.Unlock()
		} else {
			_, err = f.WriteString("#userid:username\n87784167131066368:Xackery #aka Xackery#3764")
		}

		if err != nil {
			return fmt.Errorf("create user database: %w", err)
		}
		return nil
	}

	err = reload()
	if err != nil {
		return fmt.Errorf("reload: %w", err)
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("newWatcher: %w", err)
	}

	err = watcher.Add(config.UsersDatabasePath)
	if err != nil {
		return fmt.Errorf("watcherAdd: %w", err)
	}

	go loop(watcher)
	return nil
}

func loop(watcher *fsnotify.Watcher) {
	if !isStarted {
		return
	}
	mu.Lock()
	isStarted = true
	mu.Unlock()

	defer watcher.Close()
	var err error

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				tlog.Warn("[userdb] failed to read file")
				return
			}
			if event.Op&fsnotify.Write != fsnotify.Write {
				continue
			}
			tlog.Debugf("[userdb] modified, reloading")
			err = reload()
			if err != nil {
				tlog.Warnf("[userdb] reload failed, ignoring: %s", err)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			tlog.Warnf("[userdb] read failed, ignoring: %s", err)
		}
	}
}

func reload() error {
	mu.Lock()
	defer mu.Unlock()

	ue := make(map[string]UserEntry)
	_, err := os.Stat(usersDatabasePath)
	if os.IsNotExist(err) {
		return fmt.Errorf("%s reload failed: %w", usersDatabasePath, err)
	}

	ext := filepath.Ext(usersDatabasePath)
	if ext == ".toml" {
		_, err = toml.DecodeFile(usersDatabasePath, &ue)
		if err != nil {
			return fmt.Errorf("decode toml: %w", err)
		}
	} else {
		data, err := os.ReadFile(usersDatabasePath)
		if err != nil {
			return fmt.Errorf("readFile (txt): %w", err)
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.Split(line, ":")
			if len(parts) != 2 {
				continue
			}

			discordID := strings.TrimSpace(parts[0])
			characterName := strings.TrimSpace(parts[1])
			if strings.Contains(characterName, "#") {
				characterName = strings.TrimSpace(characterName[:strings.Index(characterName, "#")])
			}

			ue[discordID] = UserEntry{
				DiscordID:     discordID,
				CharacterName: characterName,
			}
		}
	}

	users = ue
	return nil
}

// Set updates or adds an entry for a specified user id
func Set(discordID string, characterName string) {
	mu.Lock()

	ue, ok := users[discordID]
	if ok {
		ue.CharacterName = characterName
		ue.DiscordID = discordID
		return
	}

	ue = UserEntry{
		DiscordID:     discordID,
		CharacterName: characterName,
	}
	users[discordID] = ue
	err := save()
	if err != nil {
		tlog.Warnf("[userdb] save failed: %s", err)
	}
}

// Name returns the name of a user based on their ID
func Name(discordID string) string {
	var name string
	mu.RLock()
	ue, ok := users[discordID]
	if ok {
		name = ue.CharacterName
	}
	mu.RUnlock()
	return name
}

func save() error {
	f, err := os.Create(usersDatabasePath)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	defer f.Close()
	enc := toml.NewEncoder(f)
	err = enc.Encode(users)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}
	return nil
}
