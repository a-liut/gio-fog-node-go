package main

import (
	"fmt"
	"gio-fog-node/pkg/gio"

	"os"
	"os/signal"
	"syscall"
)

var stopChan = make(chan os.Signal, 1)

func main() {
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	var ble gio.Transport
	ble = gio.CreateBLETransport()

	runner := gio.NewDefaultTransportRunner()
	runner.Add(ble)

	err := runner.Run()
	if err != nil {
		panic(err)
	}

	fmt.Println("Runner started")

	<-stopChan

	// Teardown
	err = runner.Stop()
	if err != nil {
		panic(err)
	}

	fmt.Println("Runner stopped")

	fmt.Println("Done")
}
