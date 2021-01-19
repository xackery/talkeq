package database

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/jbsmith7741/toml"
	"github.com/pkg/errors"
	"github.com/xackery/log"
	"github.com/xackery/talkeq/config"
)

// UserManager manages the users registered
type UserManager struct {
	ctx               context.Context
	users             map[string]string
	mutex             sync.RWMutex
	usersDatabasePath string
	db                UserDatabase
}

// UserDatabase represents the root database object
type UserDatabase struct {
	Users map[string]UserEntry
}

// UserEntry represents a record in the database
type UserEntry struct {
	CharacterName string
	DiscordID     string
}

// NewUserManager instantiates a new users database manager
func NewUserManager(ctx context.Context, config *config.Config) (*UserManager, error) {
	u := new(UserManager)
	u.usersDatabasePath = config.UsersDatabasePath
	u.db.Users = make(map[string]UserEntry)

	err := u.reloadDatabase()
	if err != nil {
		return nil, errors.Wrap(err, "reloadDatabase")
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, errors.Wrap(err, "newwatcher")
	}

	err = watcher.Add(config.UsersDatabasePath)
	if err != nil {
		return nil, errors.Wrap(err, "watcheradd")
	}

	go u.loop(ctx, watcher)
	return u, nil
}

func (u *UserManager) loop(ctx context.Context, watcher *fsnotify.Watcher) {
	log := log.New()
	defer watcher.Close()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				log.Warn().Msg("user database failed to read file")
				return
			}
			if event.Op&fsnotify.Write != fsnotify.Write {
				continue
			}
			log.Debug().Msg("users database modified, reloading")
			err := u.reloadDatabase()
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

func (u *UserManager) reloadDatabase() error {

	ue := make(map[string]UserEntry)
	_, err := os.Stat(u.usersDatabasePath)
	if os.IsNotExist(err) {
		f, err := os.Create(u.usersDatabasePath)
		if err != nil {
			return fmt.Errorf("create user database: %w", err)
		}
		defer f.Close()
		enc := toml.NewEncoder(f)
		err = enc.Encode(ue)
		if err != nil {
			return fmt.Errorf("create user database: %w", err)
		}
		u.mutex.Lock()
		u.db.Users = ue
		u.mutex.Unlock()
		return nil
	}

	_, err = toml.DecodeFile(u.usersDatabasePath, &ue)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	u.mutex.Lock()
	u.db.Users = ue
	u.mutex.Unlock()
	return nil
}

// Set updates or adds an entry for a specified user id
func (u *UserManager) Set(discordID string, characterName string) {
	log := log.New()
	u.mutex.Lock()
	ue, ok := u.db.Users[discordID]
	if ok {
		ue.CharacterName = characterName
		ue.DiscordID = discordID
	} else {
		ue = UserEntry{
			DiscordID:     discordID,
			CharacterName: characterName,
		}
	}
	u.db.Users[discordID] = ue
	err := u.save()
	if err != nil {
		log.Warn().Err(err).Msg("set user entry")
	}
	u.mutex.Unlock()
}

// Name returns the name of a user based on their ID
func (u *UserManager) Name(discordID string) string {
	var name string
	u.mutex.RLock()
	ue, ok := u.db.Users[discordID]
	if ok {
		name = ue.CharacterName
	}
	u.mutex.RUnlock()
	return name
}

func (u *UserManager) save() error {
	f, err := os.Create(u.usersDatabasePath)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	defer f.Close()
	enc := toml.NewEncoder(f)
	err = enc.Encode(u.db.Users)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}
	return nil
}
