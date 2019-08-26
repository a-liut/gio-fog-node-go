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
	if err := os.Setenv("DEVICE_SERVICE_HOST", "device_service"); err != nil {
		panic(err)
	}

	if err := os.Setenv("DEVICE_SERVICE_PORT", "5001"); err != nil {
		panic(err)
	}

	checkVariables()

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

func checkVariables() {
	if deviceServiceHost := os.Getenv("DEVICE_SERVICE_HOST"); deviceServiceHost == "" {
		panic("DEVICE_SERVICE_HOST not set.")
	}
	if deviceServicePort := os.Getenv("DEVICE_SERVICE_PORT"); deviceServicePort == "" {
		panic("DEVICE_SERVICE_PORT not set.")
	}
}
