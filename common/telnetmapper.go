package common

import "context"

// TelnetMapper creates a common interface for Telnet that other services can tap into
// to communicate without cyclic imports
type TelnetMapper interface {
	WhoCache(ctx context.Context, search string) string
}
