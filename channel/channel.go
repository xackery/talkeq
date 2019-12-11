package channel

import (
	"strings"
	"sync"
)

const (
	// Auction channel name
	Auction = "auction"
	// OOC channel name
	OOC = "ooc"
	// General Chat channel name
	General = "general"
	// Guild Chat channel name
	Guild = "guild"
	// Shout channel name
	Shout = "shout"
)

var (
	// https://eqemu.gitbook.io/server/categories/types/chat-channel-types
	channels = map[string]int{
		"ooc":     260,
		"auction": 261,
		"general": 291,
		"guild":   259,
		"shout":   262,
	}
	mutex sync.RWMutex
)

// ToString converts a channel from an int to a string
func ToString(channelID int) string {
	mutex.RLock()
	defer mutex.RUnlock()
	for k, v := range channels {
		if v == channelID {
			return k
		}
	}
	return ""
}

// IsValidString returns true if channel parses
func IsValidString(channelID string) bool {
	mutex.RLock()
	defer mutex.RUnlock()
	_, ok := channels[channelID]
	return ok
}

// IsValidInt returns true if channel prarses
func IsValidInt(channelID int) bool {
	mutex.RLock()
	defer mutex.RUnlock()
	for _, v := range channels {
		if v == channelID {
			return true
		}
	}
	return false
}

// ToInt converts a channel from a string to an int
func ToInt(channelID string) int {
	mutex.RLock()
	defer mutex.RUnlock()
	v := channels[strings.ToLower(channelID)]
	return v
}
