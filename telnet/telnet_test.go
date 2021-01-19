package telnet

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xackery/log"
	"github.com/xackery/talkeq/config"
)

func TestConvertLinks(t *testing.T) {
	assert := assert.New(t)
	//[\x1200046F000000000000000000000000000000000000000Mask of Tinkering\x12]
	//latest looks like this
	//\x1200F406000000000000000000000000000000000000000000B519D6B0Ring of Prophetic Visions\x12'\n"
	client, err := New(context.Background(), config.Telnet{
		ItemURL: "http://test.com?itemid=",
	})
	if !assert.NoError(err) {
		t.Fatal(err)
	}

	messages := []string{
		`no url test`,
		"\x1200046F000000000000000000000000000000000000000Mask of Tinkering\x12 0.8.0 style",
		"\x1200F406000000000000000000000000000000000000000000B519D6B0Ring of Prophetic Visions\x12'\n",
		"\x1200046F00000000000000000000000000000000000000000014D2720CMask of Tinkering\x12",
		"multiple link test \x1200046F00000000000000000000000000000000000000000014D2720CMask of Tinkering\x12 and second \x1200046F00000000000000000000000000000000000000000014D2720CMask of Tinkering\x12",
		"\x1200046F000000000000000000000000000000000000000Mask of Tinkering\x12 0.8.0 style double link \x1200046F000000000000000000000000000000000000000Mask of Tinkering\x12",
		"\r> \b\bShin says ooc, '\x120112A4000000000000000000000000000000000000000000244AE3C6Frosted Gem of Ferocity\x12'\n",
	}
	log := log.New()
	for _, message := range messages {
		message = client.convertLinks(message)
		log.Debug().Msgf("finalMessage: %s", message)
	}
}
