package character

import (
	"fmt"
	"sync"
)

var (
	characters  = make(map[string]*Character)
	mu          sync.RWMutex
	onlineCount int
)

// Character represents a character inside EverQuest
type Character struct {
	IsOnline bool
	Name     string
}

type Characters []*Character

func (e *Character) Clone() *Character {
	return &Character{
		IsOnline: e.IsOnline,
		Name:     e.Name,
	}
}

func (e *Character) String() string {
	return fmt.Sprintf("%s", e.Name)
}

// Characters returns a list of characters
func CharactersOnline() Characters {
	mu.RLock()
	defer mu.RUnlock()
	resp := []*Character{}
	for _, p := range characters {
		if !p.IsOnline {
			continue
		}
		resp = append(resp, p.Clone())
	}
	return resp
}

func FlushCharacters() {
	mu.Lock()
	defer mu.Unlock()
	characters = make(map[string]*Character)
}

func SetCharacters(req Characters) error {
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
