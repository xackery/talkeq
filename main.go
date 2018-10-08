package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/xackery/talkeq/client"
)

func main() {

	err := start()
	if err != nil {
		fmt.Println("exited with error:", err.Error())
		os.Exit(1)
	}
	fmt.Println("exited safely")
	os.Exit(0)
}

func start() (err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	c, err := client.New(ctx)
	if err != nil {
		return
	}
	err = c.Start(ctx)
	if err != nil {
		return
	}

	select {
	case <-ctx.Done():
	case <-signalChan:
		err = fmt.Errorf("exited due to interrupt")
	}
	return
}
