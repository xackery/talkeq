package telnet

import (
	"context"
	"testing"

	"github.com/xackery/talkeq/config"
)

func TestOnline(t *testing.T) {

	type test struct {
		first  string
		second string
		third  string
		result bool
	}

	telnet, err := New(context.Background(), config.Telnet{})
	if err != nil {
		t.Fatalf("new: %s", err)
	}

	messages := []test{
		{
			first:  "Players on server:",
			second: "* GM-Impossible * [RolePlay 60 Grave Lord] Xackery (Dark Elf) <XackGuild> zone: arena LFG AccID: 2 AccName: xackery LSID: 103621 Status: 300\r\n",
			third:  "1 players online",
			result: true,
		},
		{
			first:  "Players on server:",
			second: "  * GM-Impossible * [60 Grave Lord] Xackery (Dark Elf) <XackGuild> zone: arena AccID: 2 AccName: xackery LSID: 103621 Status: 300\r\n",
			third:  "1 players online",
			result: true,
		},
		{
			first:  "Players on server:",
			second: "* GM-Impossible * [ANON 60 Grave Lord] Xackery (Dark Elf) <XackGuild> zone: arena AccID: 2 AccName: xackery LSID: 103621 Status: 300\r\n",
			third:  "1 players online",
			result: true,
		},
	}
	for _, message := range messages {

		result := telnet.parsePlayerEntries(message.first)
		if result != message.result {
			t.Fatalf("parsePlayersOnline first wanted return %t, got %t", message.result, result)
		}
		result = telnet.parsePlayerEntries(message.second)
		if result != message.result {
			t.Fatalf("parsePlayersOnline second wanted return %t, got %t", message.result, result)
		}
		result = telnet.parsePlayerEntries(message.third)
		if result != false {
			t.Fatalf("parsePlayersOnline third wanted return %t, got %t", false, result)
		}
		telnet.isPlayerDump = false
	}
}
