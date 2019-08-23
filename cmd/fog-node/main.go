package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"gio-fog-node/pkg/config"
	"gio-fog-node/pkg/gio"
	"os"
	"os/signal"
	"syscall"
)

var stopChan = make(chan os.Signal, 1)

func main() {
	configPath := flag.String("config", "config.json", "Configuration file")

	flag.Parse()

	if err := loadConfig(*configPath); err != nil {
		panic(err)
	}

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

func loadConfig(path string) error {
	file, _ := os.Open(path)
	defer file.Close()

	var conf config.Config
	if err := json.NewDecoder(file).Decode(&conf); err != nil {
		return err
	}

	if _, err := gio.NewDeviceService(conf.DeviceServiceConfig); err != nil {
		return err
	}

	return nil
}
