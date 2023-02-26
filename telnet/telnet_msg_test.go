package telnet

import (
	"context"
	"testing"
	"time"

	"github.com/xackery/talkeq/characterdb"
	"github.com/xackery/talkeq/config"
	"github.com/ziutek/telnet"
)

func TestConvertLinks(t *testing.T) {
	//[\x1200046F000000000000000000000000000000000000000Mask of Tinkering\x12]
	//latest looks like this
	//\x1200F406000000000000000000000000000000000000000000B519D6B0Ring of Prophetic Visions\x12'\n"
	client, err := New(context.Background(), config.Telnet{
		ItemURL: "http://test.com?itemid=",
	})
	if err != nil {
		t.Fatalf("new client: %s", err)
	}

	type test struct {
		input, output string
	}

	messages := []test{
		{input: `no url test`, output: "no url test"},
		{input: "\x1200046F000000000000000000000000000000000000000Mask of Tinkering\x12 0.8.0 style", output: "http://test.com?itemid=1135 (Mask of Tinkering) 0.8.0 style"},
		{input: "\x1200F406000000000000000000000000000000000000000000B519D6B0Ring of Prophetic Visions\x12\n", output: "http://test.com?itemid=62470 (Ring of Prophetic Visions)"},
		{input: "\x1200046F00000000000000000000000000000000000000000014D2720CMask of Tinkering\x12", output: "http://test.com?itemid=1135 (Mask of Tinkering)"},
		{input: "multiple link test \x1200046F00000000000000000000000000000000000000000014D2720CMask of Tinkering\x12 and second \x1200046F00000000000000000000000000000000000000000014D2720CMask of Tinkering\x12", output: "multiple link test http://test.com?itemid=1135 (Mask of Tinkering) and second http://test.com?itemid=1135 (Mask of Tinkering)"},
		{input: "\x1200046F000000000000000000000000000000000000000Mask of Tinkering\x12 0.8.0 style double link \x1200046F000000000000000000000000000000000000000Mask of Tinkering\x12", output: "http://test.com?itemid=1135 (Mask of Tinkering) 0.8.0 style double link http://test.com?itemid=1135 (Mask of Tinkering)"},
		{input: "\x1200046F000000000000000000000000000000000000000Mask of Tinkering**\x12 0.8.0 style double link \x1200046F000000000000000000000000000000000000000Mask of Tinkering**\x12", output: "http://test.com?itemid=1135 (Mask of Tinkering**) 0.8.0 style double link http://test.com?itemid=1135 (Mask of Tinkering**)"},
		{input: "\r> \b\bShin says ooc, '\x120112A4000000000000000000000000000000000000000000244AE3C6Frosted Gem of Ferocity\x12'\n", output: "> \u0008\u0008Shin says ooc, 'http://test.com?itemid=70308 (Frosted Gem of Ferocity)'"},
		{input: "\r> \b\bShin says ooc, '\x120CA2150000000000000000000000000000000000000000002A46AF7CTae Ew War Maul**\x12'\n", output: "> \u0008\u0008Shin says ooc, 'http://test.com?itemid=827925 (Tae Ew War Maul**)'"},
		{input: "\r> \b\bShin says ooc, '\x120CA2150000000000000000000000000000000000000000002A46AF7CSpell: Test\x12'\n", output: "> \u0008\u0008Shin says ooc, 'http://test.com?itemid=827925 (Spell: Test)'"},
		{input: "\r> \b\bShin says ooc, '\x1209756800000000000000000000000000000000000000000048F274D9Magmaband of Cestus Dei +1\x12'\n", output: "> \u0008\u0008Shin says ooc, 'http://test.com?itemid=619880 (Magmaband of Cestus Dei +1)'"},
		{input: "\r> \b\bShin says ooc, '\x1207A50C000000000000000000000000000000000000000000CC2F1766Infused 2 Handed Damage\x12'\n", output: "> \u0008\u0008Shin says ooc, 'http://test.com?itemid=501004 (Infused 2 Handed Damage)'"},
	}
	for _, message := range messages {
		result := client.convertLinks(message.input)
		if result != message.output {
			t.Fatalf("convertLinks %s failed: wanted '%s', got '%s'", message.input, message.output, result)
		}
	}
}

func TestTelnet_parseMessage(t *testing.T) {
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
		{name: "Test Online", fields: fields{}, args: args{}, want: true},
		{name: "TestBlank", want: true},
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
			if got := tr.parseMessage(tt.args.msg); got != tt.want {
				t.Errorf("Telnet.parseMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}
