package telnet

import (
	"context"
	"testing"
	"time"

	"github.com/xackery/talkeq/characterdb"
	"github.com/xackery/talkeq/config"
	"github.com/ziutek/telnet"
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

func TestTelnet_parsePlayerEntries(t *testing.T) {
	type fields struct {
		ctx            context.Context
		cancel         context.CancelFunc
		isConnected    bool
		config         config.Telnet
		conn           *telnet.Conn
		subscribers    []func(interface{}) error
		isNewTelnet    bool
		isInitialState bool
		isPlayerDump   bool
		lastPlayerDump time.Time
		characters     map[string]*characterdb.Character
	}
	type args struct {
		msg string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{name: "TestConnected", fields: fields{isConnected: false}, args: args{msg: "Test"}, want: false},
		{name: "TestBlank"},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &Telnet{
				ctx:            tt.fields.ctx,
				cancel:         tt.fields.cancel,
				isConnected:    tt.fields.isConnected,
				config:         tt.fields.config,
				conn:           tt.fields.conn,
				subscribers:    tt.fields.subscribers,
				isNewTelnet:    tt.fields.isNewTelnet,
				isInitialState: tt.fields.isInitialState,
				isPlayerDump:   tt.fields.isPlayerDump,
				lastPlayerDump: tt.fields.lastPlayerDump,
				characters:     tt.fields.characters,
			}
			if got := tr.parsePlayerEntries(tt.args.msg); got != tt.want {
				t.Errorf("Telnet.parsePlayerEntries() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTelnet_Who(t *testing.T) {
	type fields struct {
		ctx            context.Context
		cancel         context.CancelFunc
		isConnected    bool
		config         config.Telnet
		conn           *telnet.Conn
		subscribers    []func(interface{}) error
		isNewTelnet    bool
		isInitialState bool
		isPlayerDump   bool
		lastPlayerDump time.Time
		characters     map[string]*characterdb.Character
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int
		wantErr bool
	}{
		{name: "Test", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &Telnet{
				ctx:            tt.fields.ctx,
				cancel:         tt.fields.cancel,
				isConnected:    tt.fields.isConnected,
				config:         tt.fields.config,
				conn:           tt.fields.conn,
				subscribers:    tt.fields.subscribers,
				isNewTelnet:    tt.fields.isNewTelnet,
				isInitialState: tt.fields.isInitialState,
				isPlayerDump:   tt.fields.isPlayerDump,
				lastPlayerDump: tt.fields.lastPlayerDump,
				characters:     tt.fields.characters,
			}
			got, err := tr.Who(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Telnet.Who() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Telnet.Who() = %v, want %v", got, tt.want)
			}
		})
	}
}
