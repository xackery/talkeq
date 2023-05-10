package telnet

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xackery/talkeq/characterdb"
	"github.com/xackery/talkeq/tlog"
)

var (
	playersOnlineRegex = regexp.MustCompile("([0-9]+) players online")
	playerEntryRegex   = regexp.MustCompile(`(.*) \[([a-zA-Z]+)? ?([0-9]+) (.*)\] (.*) \((.*)\) .* zone\: (.*) AccID: (.*) AccName: (.*) LSID: (.*) Status: (.*)`)
)

func (t *Telnet) parsePlayerEntries(msg string) bool {
	var err error
	if t.isPlayerDump && time.Now().After(t.lastPlayerDump) {
		err = characterdb.SetCharacters(t.characters)
		if err != nil {
			tlog.Warnf("[telnet] setcharacters failed: %s", err)
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
			tlog.Warnf("[telnet] setcharacters playersOnline failed: %s", err)
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
			tlog.Debugf("[telnet] failed to parse %s level (%s): %s", msg, submatches[3], err)
			level = 0
		}

		acctID, err := strconv.Atoi(submatches[8])
		if err != nil {
			tlog.Debugf("[telnet] failed to parse %s acctID (%s): %s", msg, submatches[8], err)
			acctID = 0
		}

		lsID, err := strconv.Atoi(submatches[10])
		if err != nil {
			tlog.Debugf("[telnet] failed to parse %s lsID (%s): %s", msg, submatches[10], err)
			lsID = 0
		}

		status, err := strconv.Atoi(submatches[11])
		if err != nil {
			tlog.Debugf("[telnet] failed to parse %s status (%s): %s", msg, submatches[11], err)
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

	matches := playersOnlineRegex.FindAllStringSubmatch(msg, -1)
	if len(matches) == 0 { //pattern has no match, unsupported emote
		return false
	}

	if len(matches[0]) < 2 {
		tlog.Debugf("[telnet] ignored '%s' parse, no submatch for players online", msg)
		return false
	}

	online, err := strconv.Atoi(matches[0][1])
	if err != nil {
		tlog.Debugf("[telnet] ignored '%s' parse, online count not valid", msg)
		return false
	}

	characterdb.SetCharactersOnlineCount(online)

	return true
}

// Who returns number of online players
func (t *Telnet) Who(ctx context.Context) (int, error) {
	err := t.sendLn("who")
	if err != nil {
		return 0, fmt.Errorf("who request: %w", err)
	}
	time.Sleep(100 * time.Millisecond)
	t.mu.RLock()
	online := characterdb.CharactersOnlineCount()
	t.mu.RUnlock()
	return online, nil
}
