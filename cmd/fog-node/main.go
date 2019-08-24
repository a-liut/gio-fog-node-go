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
	checkVariables()

	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	var ble gio.Transport
	ble = gio.CreateBLETransport()

	runner := gio.NewDefaultTransportRunner()
	runner.Add(ble)

	if err := runner.Run(); err != nil {
		panic(err)
	}

	fmt.Println("Runner started")

	<-stopChan

	// Teardown
	if err := runner.Stop(); err != nil {
		panic(err)
	}

	fmt.Println("Runner stopped")

	fmt.Println("Done")
}

func checkVariables() {
	if deviceServiceHost := os.Getenv("DEVICE_SERVICE_HOST"); deviceServiceHost == "" {
		panic("DEVICE_SERVICE_HOST not set.")
	}
	if deviceServicePort := os.Getenv("DEVICE_SERVICE_PORT"); deviceServicePort == "" {
		panic("DEVICE_SERVICE_PORT not set.")
	}
}
