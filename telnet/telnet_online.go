package telnet

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/xackery/log"
	"github.com/xackery/talkeq/characterdb"
)

var (
	playersOnlineRegex = regexp.MustCompile("([0-9]+) players online")
	playerEntryRegex   = regexp.MustCompile(`(.*) \[([a-zA-Z]+)? ?([0-9]+) (.*)\] (.*) \((.*)\) .* zone\: (.*) AccID: (.*) AccName: (.*) LSID: (.*) Status: (.*)`)
)

func (t *Telnet) parsePlayerEntries(msg string) bool {
	var err error
	log := log.New()
	if t.isPlayerDump && time.Now().After(t.lastPlayerDump) {
		err = characterdb.SetCharacters(t.characters)
		if err != nil {
			log.Error().Err(err).Msgf("setcharacters")
			return true
		}
		t.isPlayerDump = false
		return false
	}
	if !t.isPlayerDump && strings.Contains(msg, "Players on server:") {
		t.isPlayerDump = true
		t.lastPlayerDump = time.Now().Add(1 * time.Second)
		t.characters = make(map[string]*characterdb.Character)
		return true
	}
	if !t.isPlayerDump {
		return false
	}

	if t.isPlayerDump && strings.Contains(msg, "players online") {
		err = characterdb.SetCharacters(t.characters)
		if err != nil {
			log.Error().Err(err).Msgf("setcharacters")
			return true
		}
		t.isPlayerDump = false
		return false
	}

	matches := playerEntryRegex.FindAllStringSubmatch(strings.ReplaceAll(msg, "\r", ""), -1)
	if len(matches) == 0 {
		return false
	}

	for _, submatches := range matches {
		if len(submatches) < 6 {
			continue
		}

		level, err := strconv.Atoi(submatches[3])
		if err != nil {
			log.Debug().Err(err).Msgf("[telnet] failed to parse %s level (%s)", msg, submatches[2])
			level = 0
		}

		acctID, err := strconv.Atoi(submatches[8])
		if err != nil {
			log.Debug().Err(err).Msgf("[telnet] failed to parse %s acctID (%s)", msg, submatches[7])
			acctID = 0
		}

		lsID, err := strconv.Atoi(submatches[10])
		if err != nil {
			log.Debug().Err(err).Msgf("[telnet] failed to parse %s lsID (%s)", msg, submatches[9])
			lsID = 0
		}

		status, err := strconv.Atoi(submatches[11])
		if err != nil {
			log.Debug().Err(err).Msgf("[telnet] failed to parse %s status (%s)", msg, submatches[10])
			status = 0
		}
		t.characters[submatches[5]] = &characterdb.Character{
			IsOnline: true,
			Identity: submatches[1],
			State:    submatches[2],
			Level:    level,
			Class:    submatches[4],
			Name:     submatches[5],
			Race:     submatches[6],
			Zone:     submatches[7],
			AcctID:   acctID,
			AcctName: submatches[9],
			LSID:     lsID,
			Status:   status,
		}
	}

	return true
}

func (t *Telnet) parsePlayersOnline(msg string) bool {
	log := log.New()

	matches := playersOnlineRegex.FindAllStringSubmatch(msg, -1)
	if len(matches) == 0 { //pattern has no match, unsupported emote
		return false
	}

	if len(matches[0]) < 2 {
		log.Debug().Str("msg", msg).Msg("ignored, no submatch for players online")
		return false
	}

	online, err := strconv.Atoi(matches[0][1])
	if err != nil {
		log.Debug().Str("msg", msg).Msg("online count ignored, parse failed")
		return false
	}

	characterdb.SetCharactersOnlineCount(online)

	return true
}

// Who returns number of online players
func (t *Telnet) Who(ctx context.Context) (int, error) {
	err := t.sendLn("who")
	if err != nil {
		return 0, errors.Wrap(err, "who request")
	}
	time.Sleep(100 * time.Millisecond)
	t.mu.RLock()
	online := characterdb.CharactersOnlineCount()
	t.mu.RUnlock()
	return online, nil
}
