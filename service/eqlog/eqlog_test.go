package eqlog

import (
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEQLog(t *testing.T) {
	d := &EQLog{}
	assert.EqualError(t, d.Initialize(), "log not initialized")
	d.Log = log.New(os.Stdout, "[EQLog]", 0)
	assert.EqualError(t, d.Initialize(), "file to watch not specified")
	d.File = "foo"
	assert.NoError(t, d.Initialize())
}
