package characterdb

import (
	"fmt"
	"strings"
	"sync"

	"github.com/xackery/log"
)

var (
	characters  = make(map[string]*Character)
	mu          sync.RWMutex
	onlineCount int
)

// Character represents a character inside EverQuest
type Character struct {
	IsOnline bool
	Identity string
	State    string
	Level    int
	Class    string
	Name     string
	Race     string
	Zone     string
	AcctID   int
	AcctName string
	LSID     int
	Status   int
}

type Characters []*Character

// Characters returns a string of characters
func CharactersOnline(filter string) string {
	mu.RLock()
	defer mu.RUnlock()
	content := ""
	log := log.New()

	log.Debug().Msgf("iterating players (%d total) with filter '%s'", len(characters), filter)
	totalCount := 0
	hiddenCount := 0
	isTruncated := false
	for _, user := range characters {
		if totalCount >= 20 {
			isTruncated = true
		}
		if strings.Contains(user.State, "ANON") {
			hiddenCount++
			continue
		}
		if strings.Contains(user.State, "RolePlay") {
			hiddenCount++
			continue
		}
		/*if user.Status > 0 {
			hiddenCount++
			continue
		}*/

		if filter == "" {
			content += fmt.Sprintf("%s\n", user.Name)
			totalCount++
			continue
		}

		if !strings.Contains(user.Name, filter) &&
			!strings.Contains(user.Zone, filter) {
			continue
		}

		content += fmt.Sprintf("%s\n", user.Name)
		totalCount++
	}

	hiddenText := ""
	if hiddenCount > 0 {
		hiddenText = "(%d hidden) "
	}

	truncatedText := ""
	if isTruncated {
		truncatedText = "(truncated) "
	}

	if totalCount == 0 {
		content = fmt.Sprintf("There are 0 players %sonline.", hiddenText)
		return content
	}
	if filter == "" {
		content = fmt.Sprintf("There are %d players %sonline%s:\n%s", totalCount, hiddenText, truncatedText, content)
		return content
	}

	content = fmt.Sprintf("There are %d players %s%swho match '%s':\n%s", totalCount, hiddenText, truncatedText, filter, content)
	return content
}

func SetCharacters(req map[string]*Character) error {
	mu.Lock()
	defer mu.Unlock()
	log := log.New()
	log.Debug().Msgf("setting characters to %d", len(characters))
	characters = req
	onlineCount = len(characters)
	return nil
}

func CharactersOnlineCount() int {
	mu.RLock()
	defer mu.RUnlock()
	return onlineCount
}

func SetCharactersOnlineCount(value int) {
	mu.Lock()
	defer mu.Unlock()
	onlineCount = value
}
