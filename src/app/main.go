package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	
	"gio"
)

var stopChan = make(chan os.Signal, 1)

func main() {
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
	
	var ble gio.Transport
	ble = gio.CreateBLETransport()
	
	runner := gio.NewDefaultTransportRunner()
	runner.Add(ble)
	
	runner.Run()
	fmt.Println("Runner started")
	
	<-stopChan
	
	// Teardown
	runner.Stop()
	fmt.Println("Runner stopped")
	
	fmt.Println("Done")
}
