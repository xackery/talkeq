package userdb

import (
	"fmt"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/jbsmith7741/toml"
	"github.com/xackery/log"
	"github.com/xackery/talkeq/config"
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

	log := log.New()
	log.Debug().Msgf("initializing user db")
	_, err := os.Stat(usersDatabasePath)
	if os.IsNotExist(err) {
		log.Debug().Msgf("user db not found, creating a new one")
		f, err := os.Create(usersDatabasePath)
		if err != nil {
			return fmt.Errorf("create user database: %w", err)
		}
		defer f.Close()
		enc := toml.NewEncoder(f)
		mu.Lock()
		err = enc.Encode(users)
		mu.Unlock()

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
	log := log.New()

	defer watcher.Close()
	var err error

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				log.Warn().Msg("user database failed to read file")
				return
			}
			if event.Op&fsnotify.Write != fsnotify.Write {
				continue
			}
			log.Debug().Msg("users database modified, reloading")
			err = reload()
			if err != nil {
				log.Warn().Err(err).Msg("failed to reload users database")
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Warn().Err(err).Msg("user database failed to read file")
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

	_, err = toml.DecodeFile(usersDatabasePath, &ue)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	users = ue
	return nil
}

// Set updates or adds an entry for a specified user id
func Set(discordID string, characterName string) {
	mu.Lock()
	log := log.New()

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
		log.Warn().Err(err).Msg("set user entry")
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
