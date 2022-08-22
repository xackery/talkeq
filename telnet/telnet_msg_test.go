package telnet

import (
	"context"
	"testing"

	"github.com/xackery/talkeq/config"
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
		{input: "\r> \b\bShin says ooc, '\x120112A4000000000000000000000000000000000000000000244AE3C6Frosted Gem of Ferocity\x12'\n", output: "> Shin says ooc, 'http://test.com?itemid=70308 (Frosted Gem of Ferocity)'"},
		{input: "\r> \b\bShin says ooc, '\x120CA2150000000000000000000000000000000000000000002A46AF7CTae Ew War Maul**\x12'\n", output: "> Shin says ooc, 'http://test.com?itemid=827925 (Tae Ew War Maul**)'"},
		{input: "\r> \b\bShin says ooc, '\x120CA2150000000000000000000000000000000000000000002A46AF7CSpell: Test\x12'\n", output: "> Shin says ooc, 'http://test.com?itemid=827925 (Spell: Test)'"},
	}
	for _, message := range messages {
		result := client.convertLinks(message.input)
		if result != message.output {
			t.Fatalf("convertLinks %s failed: wanted %s, got %s", message.input, message.output, result)
		}
	}
}
