package database

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/jbsmith7741/toml"
	"github.com/pkg/errors"
	"github.com/xackery/log"
	"github.com/xackery/talkeq/config"
)

// RegisterManager manages !register requests
type RegisterManager struct {
	ctx                      context.Context
	db                       RegisterDatabase
	mutex                    sync.RWMutex
	RegistrationDatabasePath string
}

// RegisterDatabase wraps registrations
type RegisterDatabase struct {
	Registrations map[string]RegisterEntry `toml:"registrations"`
}

// RegisterEntry is a registration request entry
type RegisterEntry struct {
	DiscordID     string
	DiscordName   string
	Status        string
	CharacterName string
	ChannelID     string
	MessageID     string
	Timeout       int64
	Code          string
}

// NewRegisterManager instantiates a new registers database manager
func NewRegisterManager(ctx context.Context, config *config.API) (*RegisterManager, error) {
	u := new(RegisterManager)
	u.RegistrationDatabasePath = config.APIRegister.RegistrationDatabasePath
	u.db.Registrations = make(map[string]RegisterEntry)

	err := u.reloadDatabase()
	if err != nil {
		return nil, errors.Wrap(err, "reloadDatabase")
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, errors.Wrap(err, "newwatcher")
	}

	err = watcher.Add(config.APIRegister.RegistrationDatabasePath)
	if err != nil {
		return nil, errors.Wrap(err, "watcheradd")
	}

	go u.loop(ctx, watcher)
	return u, nil
}

func (u *RegisterManager) loop(ctx context.Context, watcher *fsnotify.Watcher) {
	log := log.New()
	defer watcher.Close()
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				log.Warn().Msg("register database failed to read file")
				return
			}
			if event.Op&fsnotify.Write != fsnotify.Write {
				continue
			}
			log.Debug().Msg("registers database modified, reloading")
			err := u.reloadDatabase()
			if err != nil {
				log.Warn().Err(err).Msg("failed to reload registers database")
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Warn().Err(err).Msg("register database failed to read file")
		}
	}
}

func (u *RegisterManager) reloadDatabase() error {

	nre := make(map[string]RegisterEntry)
	_, err := os.Stat(u.RegistrationDatabasePath)
	if os.IsNotExist(err) {
		f, err := os.Create(u.RegistrationDatabasePath)
		if err != nil {
			return fmt.Errorf("create register database: %w", err)
		}
		defer f.Close()
		enc := toml.NewEncoder(f)
		err = enc.Encode(nre)
		if err != nil {
			return fmt.Errorf("create register database: %w", err)
		}
		u.mutex.Lock()
		u.db.Registrations = nre
		u.mutex.Unlock()
		return nil
	}

	_, err = toml.DecodeFile(u.RegistrationDatabasePath, &nre)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	u.mutex.Lock()
	u.db.Registrations = nre
	u.mutex.Unlock()
	return nil
}

func (u *RegisterManager) save() error {
	f, err := os.Create(u.RegistrationDatabasePath)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	defer f.Close()
	enc := toml.NewEncoder(f)
	err = enc.Encode(u.db.Registrations)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}
	return nil
}

// Set updates or adds an entry for a specified register id
func (u *RegisterManager) Set(discordID string, discordName string, characterName string, channelID string, messageID string, status string, timeout int64) {
	u.mutex.Lock()
	re := RegisterEntry{
		DiscordID:     discordID,
		DiscordName:   discordName,
		CharacterName: strings.Title(characterName),
		Status:        status,
		MessageID:     messageID,
		ChannelID:     channelID,
		Timeout:       timeout,
		Code:          "1234",
	}
	u.db.Registrations[discordID] = re
	u.save()
	u.mutex.Unlock()
}

// FindByCode returns an entry if code matches and is valid
func (u *RegisterManager) FindByCode(code string) (entry RegisterEntry, err error) {
	u.mutex.RLock()
	defer u.mutex.RUnlock()
	var re RegisterEntry
	for _, entry := range u.db.Registrations {
		if entry.Code != code {
			continue
		}
		if entry.Timeout < time.Now().Unix() {
			return re, fmt.Errorf("code expired")
		}

		return entry, nil
	}
	return re, fmt.Errorf("invalid code")
}

// QueuedEntries returns a list of items that need to be relayed in EQ
func (u *RegisterManager) QueuedEntries() (entries []RegisterEntry, err error) {

	u.mutex.Lock()
	defer u.mutex.Unlock()
	for _, entry := range u.db.Registrations {
		if entry.Timeout < time.Now().Unix() {
			continue
		}
		if entry.Status != "In Queue" {
			continue
		}

		entry.Status = "Waiting Reply"
		u.db.Registrations[entry.DiscordID] = entry
		entries = append(entries, entry)
	}
	return entries, nil
}

// Entry returns an existing entry
func (u *RegisterManager) Entry(discordID string) (RegisterEntry, error) {
	u.mutex.RLock()
	defer u.mutex.RUnlock()
	re, ok := u.db.Registrations[discordID]
	if !ok {
		return re, fmt.Errorf("not found")
	}
	return re, nil
}

// CharacterName returns the name of a discord ID
func (u *RegisterManager) CharacterName(discordID string) string {
	u.mutex.RLock()
	defer u.mutex.RUnlock()
	re, ok := u.db.Registrations[discordID]
	if !ok {
		return ""
	}
	if re.Status != "confirmed" {
		return ""
	}
	return re.CharacterName
}

// Update sets a new timeout and status to a registration entry
func (u *RegisterManager) Update(discordID string, status string, timeout int64) error {
	u.mutex.Lock()
	defer u.mutex.Unlock()
	re, ok := u.db.Registrations[discordID]
	if !ok {
		return fmt.Errorf("discordID %s not found", discordID)
	}
	re.Status = status
	re.Timeout = timeout
	if re.Status == "Confirmed" {
		re.Code = ""
	}
	u.db.Registrations[discordID] = re
	err := u.save()
	if err != nil {
		return fmt.Errorf("save: %w", err)
	}
	return nil
}
