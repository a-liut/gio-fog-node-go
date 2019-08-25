package main

import (
	"encoding/json"
	"fmt"
	"gio-fog-node/pkg/gio"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

const (
	serverDefaultPort = "5003"
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

	log.Println("Runner started")

	go runServer(ble.(*gio.BLETransport))

	<-stopChan

	// Teardown
	if err := runner.Stop(); err != nil {
		panic(err)
	}

	log.Println("Runner stopped")

	log.Println("Done")
}

func runServer(transport *gio.BLETransport) {
	r := mux.NewRouter()

	// Register endpoints
	r.HandleFunc("/devices", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Requested device list")

		devices := transport.GetDevices()

		log.Printf("Devices: %v\n", devices)

		err := json.NewEncoder(w).Encode(devices)
		if err != nil {
			log.Println(err)
		}
	})

	r.HandleFunc("/devices/{deviceId}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		deviceId := vars["deviceId"]

		log.Printf("Requested device %s information", deviceId)

		d := transport.GetDeviceByID(deviceId)

		data, _ := json.Marshal(d)

		_, err := fmt.Fprintf(w, string(data))
		if err != nil {
			log.Println(err)
		}
	})

	// Start server
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = serverDefaultPort
	}
	port = fmt.Sprintf(":%s", port)

	log.Println(http.ListenAndServe(port, r))
}

func checkVariables() {
	if deviceServiceHost := os.Getenv("DEVICE_SERVICE_HOST"); deviceServiceHost == "" {
		panic("DEVICE_SERVICE_HOST not set.")
	}
	if deviceServicePort := os.Getenv("DEVICE_SERVICE_PORT"); deviceServicePort == "" {
		panic("DEVICE_SERVICE_PORT not set.")
	}
}
