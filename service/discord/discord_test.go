package discord

import (
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xackery/discordeq/model"
)

func TestDiscord(t *testing.T) {
	d := &Discord{}
	assert.EqualError(t, d.Initialize(), "log not initialized")
	d.Log = log.New(os.Stdout, "[Discord]", 0)
	assert.EqualError(t, d.Initialize(), "Server ID not configured")
	d.ServerID = "1234"
	errAuth := model.ErrAuth{}
	assert.EqualError(t, d.Initialize(), errAuth.Error())
	d.Token = "1234"
	assert.NoError(t, d.Initialize())
}
