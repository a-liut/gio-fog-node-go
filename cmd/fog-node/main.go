/*
 * Fog Node
 *
 * A tool for connecting devices to the Giò Plants platform.
 *
 * API version: 1.0.0
 * Contact: andrea.liut@gmail.com
 */
package main

import (
	"gio-fog-node/pkg/gio"
	"log"
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

	if err := runner.Run(); err != nil {
		panic(err)
	}

	log.Println("Runner started")

	go gio.RunServer(ble.(*gio.BLETransport))

	<-stopChan

	// Teardown
	if err := runner.Stop(); err != nil {
		panic(err)
	}

	log.Println("Runner stopped")

	log.Println("Done")
}
