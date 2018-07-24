package nats

import (
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNats(t *testing.T) {
	n := &Nats{}
	assert.EqualError(t, n.Initialize(), "log not initialized")
	n.Log = log.New(os.Stdout, "[NATS]", 0)
	assert.EqualError(t, n.Initialize(), "failed to connect to nats: nats: no servers available for connection")
}
