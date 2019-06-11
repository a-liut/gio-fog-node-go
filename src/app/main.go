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
	
	runner := gio.NewDefaultTransportRunner(stopChan)
	go func() {
		runner.Add(ble)
		
		runner.Run()
	}()
	
	<-stopChan
	
	// Teardown
	runner.Stop()
	
	fmt.Println("Done")
}
