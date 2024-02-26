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
	client.config.IsLegacyLinks = true

	type test struct {
		name   string
		input  string
		output string
	}

	messages := []test{
		{
			name:   "mask of tinkering x64",
			input:  "\x120000027180000000000000000000000000000000000000000000000000000000000003271C223Gold Ring (Latent)\x12",
			output: "http://test.com?itemid=10008 (Gold Ring (Latent))",
		}, {
			name:   "mask of tinkering x32",
			input:  "\r> \b\bShin says ooc, '\x1207A50C000000000000000000000000000000000000000000CC2F1766Infused 2 Handed Damage\x12'\n",
			output: "> \u0008\u0008Shin says ooc, 'http://test.com?itemid=501004 (Infused 2 Handed Damage)'",
		}, {
			name:   "no url test",
			input:  `no url test`,
			output: "no url test",
		}, {
			name:   "mask of tinkering",
			input:  "\x1200046F000000000000000000000000000000000000000Mask of Tinkering\x12 0.8.0 style",
			output: "http://test.com?itemid=1135 (Mask of Tinkering) 0.8.0 style",
		}, {
			name:   "ring of prophetic visions",
			input:  "\x1200F406000000000000000000000000000000000000000000B519D6B0Ring of Prophetic Visions\x12\n",
			output: "http://test.com?itemid=62470 (Ring of Prophetic Visions)",
		}, {
			name:   "mask of tinkering",
			input:  "\x1200046F00000000000000000000000000000000000000000014D2720CMask of Tinkering\x12",
			output: "http://test.com?itemid=1135 (Mask of Tinkering)",
		}, {
			name:   "multiple link test",
			input:  "multiple link test \x1200046F00000000000000000000000000000000000000000014D2720CMask of Tinkering\x12 and second \x1200046F00000000000000000000000000000000000000000014D2720CMask of Tinkering\x12",
			output: "multiple link test http://test.com?itemid=1135 (Mask of Tinkering) and second http://test.com?itemid=1135 (Mask of Tinkering)",
		}, {
			name:   "0.8.0 style double link",
			input:  "\x1200046F000000000000000000000000000000000000000Mask of Tinkering\x12 0.8.0 style double link \x1200046F000000000000000000000000000000000000000Mask of Tinkering\x12",
			output: "http://test.com?itemid=1135 (Mask of Tinkering) 0.8.0 style double link http://test.com?itemid=1135 (Mask of Tinkering)",
		}, {
			name:   "0.8.0 style double link with asterisk",
			input:  "\x1200046F000000000000000000000000000000000000000Mask of Tinkering**\x12 0.8.0 style double link \x1200046F000000000000000000000000000000000000000Mask of Tinkering**\x12",
			output: "http://test.com?itemid=1135 (Mask of Tinkering**) 0.8.0 style double link http://test.com?itemid=1135 (Mask of Tinkering**)",
		}, {
			name:   "forested gem",
			input:  "\r> \b\bShin says ooc, '\x120112A4000000000000000000000000000000000000000000244AE3C6Frosted Gem of Ferocity\x12'\n",
			output: "> \u0008\u0008Shin says ooc, 'http://test.com?itemid=70308 (Frosted Gem of Ferocity)'",
		}, {
			name:   "tae ew war maul",
			input:  "\r> \b\bShin says ooc, '\x120CA2150000000000000000000000000000000000000000002A46AF7CTae Ew War Maul**\x12'\n",
			output: "> \u0008\u0008Shin says ooc, 'http://test.com?itemid=827925 (Tae Ew War Maul**)'",
		}, {
			name:   "spell test",
			input:  "\r> \b\bShin says ooc, '\x120CA2150000000000000000000000000000000000000000002A46AF7CSpell: Test\x12'\n",
			output: "> \u0008\u0008Shin says ooc, 'http://test.com?itemid=827925 (Spell: Test)'",
		}, {
			name:   "cestus test",
			input:  "\r> \b\bShin says ooc, '\x1209756800000000000000000000000000000000000000000048F274D9Magmaband of Cestus Dei +1\x12'\n",
			output: "> \u0008\u0008Shin says ooc, 'http://test.com?itemid=619880 (Magmaband of Cestus Dei +1)'",
		}, {
			name:   "2 handed test1",
			input:  "\r> \b\bShin says ooc, '\x1207A50C000000000000000000000000000000000000000000CC2F1766Infused 2 Handed Damage\x12'\n",
			output: "> \u0008\u0008Shin says ooc, 'http://test.com?itemid=501004 (Infused 2 Handed Damage)'",
		}, {
			name:   "2 handed test2",
			input:  "\r> \b\bShin says ooc, '\x1207A50C000000000000000000000000000000000000000000CC2F1766Infused 2 Handed Damage\x12'\n",
			output: "> \u0008\u0008Shin says ooc, 'http://test.com?itemid=501004 (Infused 2 Handed Damage)'",
			//		}, {
			//			input:  "\r> \b\bShin says ooc, '\x1201E2A30000000000000000000000000000000000000000007BEA6B30Spell: Wild Cat V (Tier 9)\x12'\n",
			//			output: "> \u0008\u0008Shin says ooc, 'http://test.com?itemid=62470 (Spell: Wild Cat V (Tier 9))",
			//}, {
			//	input:  "\r> \b\bShin says ooc, '\x1201E2A30000000000000000000000000000000000000000007BEA6B30Spell: Wild Cat V (Tier 9)\x12, \x1201E29E0000000000000000000000000000000000000000001BE3AE22Spell: Maelstrom of the Phoenix (Tier 9)\x12, \x12013AEA000000000000000000000000000000000000000000BAEEB588Staff of Screaming Souls (Tier 9)\x12, \x12013AE400000000000000000000000000000000000000000043BE53D9Cryptwood Truncheon (Tier 9)\x12 pst rank 7'\n",
			//	output: "> \u0008\u0008Shin says ooc, 'http://test.com?itemid=62470 (Spell: Wild Cat V (Tier 9)",
		},
	}
	for _, message := range messages {
		result := client.convertLinks(message.input)
		if result != message.output {
			t.Fatalf("convertLinks %s failed: got %s, wanted %s", message.name, result, message.output)
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
