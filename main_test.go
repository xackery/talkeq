package main

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/xackery/talkeq/service"
	"github.com/xackery/talkeq/service/discord"
	"github.com/xackery/talkeq/service/eqlog"
	"github.com/xackery/talkeq/service/telnet"
)

func TestMain(t *testing.T) {
	var err error
	fmt.Println("Starting")

	te := &telnet.Telnet{
		Log: log.New(os.Stdout, "[Telnet]", 0),
	}
	d := &discord.Discord{
		Log:      log.New(os.Stdout, "[Discord]", 0),
		ServerID: "123",
	}

	e := &eqlog.EQLog{
		Log:  log.New(os.Stdout, "[EQLog]", 0),
		File: "test",
	}

	switchboard := &service.Switchboard{
		Patches: []*service.Patch{
			{
				From:   te,
				To:     d,
				Number: "260",
			},
			{
				From:   d,
				To:     te,
				Number: "260",
			},
		},
	}

	d.Switchboard = switchboard
	err = d.Initialize()
	//assert.NoError(t, err)

	te.Switchboard = switchboard
	err = te.Initialize()
	//assert.NoError(t, err)

	e.Switchboard = switchboard
	err = te.Initialize()
	//assert.NoError(t, err)

	if err == nil {
		t.Fail()
	}

}
