package database

import (
	"context"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
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
}

// NewUserManager instantiates a new users database manager
func NewUserManager(ctx context.Context, config *config.Config) (*UserManager, error) {
	u := new(UserManager)
	u.usersDatabasePath = config.UsersDatabasePath
	u.users = make(map[string]string)

	_, err := os.Stat(u.usersDatabasePath)
	if os.IsNotExist(err) {
		err = ioutil.WriteFile(u.usersDatabasePath, []byte(`# userid:username
87784167131066368:Xackery # aka Xackery#3764`), 0644)
		if err != nil {
			return nil, errors.Wrapf(err, "users database create %s", u.usersDatabasePath)
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
	log := log.New()
	data, err := ioutil.ReadFile(u.usersDatabasePath)
	if err != nil {
		return errors.Wrap(err, "reloadDatabase")
	}

	nu := make(map[string]string)
	lines := strings.Split(string(data), "\n")
	for lineNumber, line := range lines {
		lineNumber++
		p := strings.Index(line, ":")
		if p < 1 {
			log.Warn().Int("line number", lineNumber).Msgf("%s no : exists", u.usersDatabasePath)
			continue
		}
		id := line[0:p]
		if len(id) < 3 {
			log.Warn().Int("line number", lineNumber).Msgf("%s userid too short", u.usersDatabasePath)
			continue
		}
		name := line[p+1:]
		if len(name) < 3 {
			log.Warn().Int("line number", lineNumber).Msgf("%s username too short", u.usersDatabasePath)
			continue
		}
		p = strings.Index(name, "#")
		if p > 0 {
			name = name[0:p]
		}
		name = strings.TrimSpace(name)
		_, ok := nu[id]
		if ok {
			log.Warn().Int("line number", lineNumber).Str("user id", id).Msgf("%s duplicate entry", u.usersDatabasePath)
		}
		nu[id] = name
	}

	u.mutex.Lock()
	u.users = nu
	u.mutex.Unlock()
	return nil
}

// Set updates or adds an entry for a specified user id
func (u *UserManager) Set(userID string, userName string) {
	u.mutex.Lock()
	u.users[userID] = userName
	u.mutex.Unlock()
}

// Name returns the name of a user based on their ID
func (u *UserManager) Name(userID string) string {
	u.mutex.RLock()
	name := u.users[userID]
	u.mutex.RUnlock()
	return name
}
