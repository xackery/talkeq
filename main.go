package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"

	"github.com/xackery/talkeq/client"
	"github.com/xackery/talkeq/tlog"
)

// Version is the build version
var Version string

func main() {
	w, err := os.Create("talkeq.log")
	if err != nil {
		fmt.Println(err)
		if runtime.GOOS == "windows" {
			option := ""
			fmt.Println("press a key then enter to exit.")
			fmt.Scan(&option)
		}
		os.Exit(1)
	}
	defer w.Close()
	tlog.Init(w, os.Stdout)

	err = run(w)
	if err != nil {
		tlog.Errorf("run failed with error: %s", err)
		if runtime.GOOS == "windows" {
			option := ""
			fmt.Println("press a key then enter to exit.")
			fmt.Scan(&option)
		}
		tlog.Sync()
		os.Exit(1)
	}
	tlog.Infof("exited safely")
	tlog.Sync()
	os.Exit(0)
}

func run(w *os.File) (err error) {

	if Version == "" {
		Version = "1.x.x EXPERIMENTAL"
	}
	tlog.Infof("starting talkeq %s", Version)
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}

	tlog.Infof("working directory is %s", wd)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	c, err := client.New(ctx)
	if err != nil {
		return fmt.Errorf("new client: %w", err)
	}

	err = c.Connect(ctx)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	select {
	case <-ctx.Done():
	case <-signalChan:
		err = c.Disconnect(ctx)
		if err != nil {
			return fmt.Errorf("signal disconnect: %w", err)
		}
		tlog.Infof("exiting, interrupt signal sent")
	}
	return
}
