package telnet

import (
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTelnet(t *testing.T) {
	d := &Telnet{}
	assert.EqualError(t, d.Initialize(), "log not initialized")
	d.Log = log.New(os.Stdout, "[Telnet]", 0)
	assert.EqualError(t, d.Initialize(), "dial tcp :0: connect: can't assign requested address: authentication failed")
}
