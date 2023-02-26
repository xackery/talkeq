package telnet

import (
	"context"
	"testing"
	"time"

	"github.com/xackery/talkeq/characterdb"
	"github.com/xackery/talkeq/config"
	"github.com/ziutek/telnet"
)

func TestTelnet_IsConnected(t *testing.T) {
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
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{name: "TestBlank"},
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
			if got := tr.IsConnected(); got != tt.want {
				t.Errorf("Telnet.IsConnected() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTelnet_Connect(t *testing.T) {
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
		wantErr bool
	}{
		{name: "Test"},
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
			if err := tr.Connect(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Telnet.Connect() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
