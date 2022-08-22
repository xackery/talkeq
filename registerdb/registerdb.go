package registerdb

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/jbsmith7741/toml"
	"github.com/xackery/log"
	"github.com/xackery/talkeq/config"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	isStarted                bool
	db                       RegisterDatabase
	mu                       sync.RWMutex
	registrationDatabasePath string
)

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

// New instantiates a new registerdb
func New(config *config.API) error {
	if isStarted {
		return fmt.Errorf("already started")
	}
	log := log.New()
	log.Debug().Msgf("initializing register db")
	registrationDatabasePath = config.APIRegister.RegistrationDatabasePath
	db.Registrations = make(map[string]RegisterEntry)

	_, err := os.Stat(registrationDatabasePath)
	if os.IsNotExist(err) {
		f, err := os.Create(registrationDatabasePath)
		if err != nil {
			return fmt.Errorf("create register database: %w", err)
		}
		defer f.Close()
		enc := toml.NewEncoder(f)
		err = enc.Encode(db.Registrations)
		if err != nil {
			return fmt.Errorf("create register database: %w", err)
		}
		err = save()
		if err != nil {
			return fmt.Errorf("save: %w", err)
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

	err = watcher.Add(registrationDatabasePath)
	if err != nil {
		return fmt.Errorf("watcher.Add: %w", err)
	}

	go loop(watcher)
	return nil
}

func loop(watcher *fsnotify.Watcher) {
	defer watcher.Close()
	log := log.New()
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				log.Warn().Msg("register database failed to read file")
				return
			}
			if event.Op&fsnotify.Write != fsnotify.Write {
				continue
			}
			continue
			/*log.Debug().Msg("registers database modified, reloading")
			err := u.reloadDatabase()
			if err != nil {
				log.Warn().Err(err).Msg("failed to reload registers database")
			}*/
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Warn().Err(err).Msg("register database failed to read file")
		}
	}
}

func reload() error {
	mu.Lock()
	defer mu.Unlock()

	nre := make(map[string]RegisterEntry)

	_, err := os.Stat(registrationDatabasePath)
	if os.IsNotExist(err) {
		return fmt.Errorf("%s does not exist: %w", registrationDatabasePath, err)
	}

	_, err = toml.DecodeFile(registrationDatabasePath, &nre)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	db.Registrations = nre
	return nil
}

func save() error {
	f, err := os.Create(registrationDatabasePath)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	defer f.Close()
	enc := toml.NewEncoder(f)
	err = enc.Encode(db.Registrations)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}
	return nil
}

// Set updates or adds an entry for a specified register id
func Set(discordID string, discordName string, characterName string, channelID string, messageID string, status string, timeout int64) {
	mu.Lock()
	defer mu.Unlock()
	log := log.New()
	re := RegisterEntry{
		DiscordID:     discordID,
		DiscordName:   discordName,
		CharacterName: cases.Title(language.AmericanEnglish).String(characterName),
		Status:        status,
		MessageID:     messageID,
		ChannelID:     channelID,
		Timeout:       timeout,
		Code:          "1234",
	}
	db.Registrations[discordID] = re
	err := save()
	if err != nil {
		log.Warn().Err(err).Msgf("save")
	}
}

// FindByCode returns an entry if code matches and is valid
func FindByCode(code string) (entry RegisterEntry, err error) {
	mu.RLock()
	defer mu.RUnlock()
	var re RegisterEntry
	for _, entry := range db.Registrations {
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
func QueuedEntries() (entries []RegisterEntry, err error) {
	mu.Lock()
	defer mu.Unlock()
	log := log.New()
	log.Debug().Int("registrations", len(db.Registrations)).Msg("[api]")
	for _, entry := range db.Registrations {
		if entry.Timeout < time.Now().Unix() {
			continue
		}
		if entry.Status != "In Queue" {
			continue
		}

		entry.Status = "Waiting Reply"
		db.Registrations[entry.DiscordID] = entry
		entries = append(entries, entry)
	}
	return entries, nil
}

// Entry returns an existing entry
func Entry(discordID string) (RegisterEntry, error) {
	mu.RLock()
	defer mu.RUnlock()
	re, ok := db.Registrations[discordID]
	if !ok {
		return re, fmt.Errorf("not found")
	}
	return re, nil
}

// CharacterName returns the name of a discord ID
func CharacterName(discordID string) string {
	mu.RLock()
	defer mu.RUnlock()
	re, ok := db.Registrations[discordID]
	if !ok {
		return ""
	}
	if re.Status != "confirmed" {
		return ""
	}
	return re.CharacterName
}

// Update sets a new timeout and status to a registration entry
func Update(discordID string, status string, timeout int64) error {
	mu.Lock()
	defer mu.Unlock()
	re, ok := db.Registrations[discordID]
	if !ok {
		return fmt.Errorf("discordID %s not found", discordID)
	}
	re.Status = status
	re.Timeout = timeout
	if re.Status == "Confirmed" {
		re.Code = ""
	}
	db.Registrations[discordID] = re
	err := save()
	if err != nil {
		return fmt.Errorf("save: %w", err)
	}
	return nil
}
