package telnet

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/xackery/log"
	"github.com/xackery/talkeq/character"
)

var (
	playersOnlineRegex = regexp.MustCompile("([0-9]+) players online")
)

func (t *Telnet) parsePlayersOnline(msg string) bool {
	log := log.New()

	matches := playersOnlineRegex.FindAllStringSubmatch(msg, -1)
	if len(matches) == 0 { //pattern has no match, unsupported emote
		return false
	}
	log.Debug().Msg("detected players online pattern")

	if len(matches[0]) < 2 {
		log.Debug().Str("msg", msg).Msg("ignored, no submatch for players online")
		return false
	}

	online, err := strconv.Atoi(matches[0][1])
	if err != nil {
		log.Debug().Str("msg", msg).Msg("online count ignored, parse failed")
		return false
	}

	character.SetCharactersOnlineCount(online)

	players := character.Characters{}

	fmt.Println(msg)
	lines := strings.Split(msg, "\n")
	for _, line := range lines {
		if strings.Contains(line, "players online") {
			continue
		}
		players = append(players, &character.Character{
			IsOnline: true,
		})
	}
	err = character.SetCharacters(players)
	if err != nil {
		log.Error().Err(err).Msgf("setcharacters")
	}

	log.Debug().Int("online", online).Msg("updated online count")
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
	online := character.CharactersOnlineCount()
	t.mu.RUnlock()
	return online, nil
}
