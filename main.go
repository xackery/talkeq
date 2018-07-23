package main

import (
	"fmt"
	"log"
	"os"

	"github.com/xackery/talkeq/service"
	"github.com/xackery/talkeq/service/discord"
	"github.com/xackery/talkeq/service/telnet"
)

func main() {
	var err error
	fmt.Println("Starting")

	t := &telnet.Telnet{
		Log: log.New(os.Stdout, "[Telnet]", 0),
	}
	d := &discord.Discord{
		Log: log.New(os.Stdout, "[Discord]", 0),
	}

	switchboard := &service.Switchboard{
		Patches: []*service.Patch{
			{
				From:    t,
				To:      d,
				ChanNum: 260,
			},
			{
				From:    d,
				To:      t,
				ChanNum: 260,
			},
		},
	}

	d.Switchboard = switchboard
	err = d.Initialize()
	if err != nil {
		fmt.Println("Failed to initialize discord", err.Error())
		return
	}
	t.Switchboard = switchboard
	err = t.Initialize()
	if err != nil {
		fmt.Println("Failed to initialize telnet", err.Error())
		return
	}

}
