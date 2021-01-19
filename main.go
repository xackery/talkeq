package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"

	"github.com/pkg/errors"
	"github.com/xackery/log"
	"github.com/xackery/talkeq/client"
)

// Version is the build version
var Version string

func main() {
	log := log.New()
	if Version == "" {
		Version = "1.x.x"
	}
	log.Info().Msgf("starting talkeq %s", Version)

	err := run()
	if err != nil {
		log.Err(err).Msg("exited with error")
		if runtime.GOOS == "windows" {
			option := ""
			fmt.Println("press a key then enter to exit.")
			fmt.Scan(&option)
		}
		os.Exit(1)
	}
	log.Info().Msg("exited safely")
	os.Exit(0)
}

func run() (err error) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	c, err := client.New(ctx)
	if err != nil {
		return errors.Wrap(err, "new client")
	}

	err = c.Connect(ctx)
	if err != nil {
		return errors.Wrap(err, "connect")
	}

	select {
	case <-ctx.Done():
	case <-signalChan:
		err = c.Disconnect(ctx)
		if err != nil {
			return errors.Wrap(err, "signal disconnect")
		}
		fmt.Println("\nexiting, interrupt signal sent")
	}
	return
}
