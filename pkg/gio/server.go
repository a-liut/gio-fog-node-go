package gio

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
)

const (
	serverDefaultPort = "5003"
)

type Endpoint struct {
	Path    string
	Handler func(w http.ResponseWriter, r *http.Request)
	Methods []string
}

type ApiResponse struct {
	Message string `json:"message"`
}

// Transport to be used
var transport *BLETransport
var endpoints = []Endpoint{
	{
		// List all connected devices
		Path: "/devices",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			log.Println("Requested device list")

			devices := transport.GetDevices()

			log.Printf("Devices: %v\n", devices)

			err := json.NewEncoder(w).Encode(devices)
			if err != nil {
				log.Println(err)
			}
		},
		Methods: []string{http.MethodGet},
	},
	{
		// Get a single connected device
		Path: "/devices/{deviceId}",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			deviceId := vars["deviceId"]

			log.Printf("Requested device %s information\n", deviceId)

			d := transport.GetDeviceByID(deviceId)
			if d == nil {
				// Not found
				w.WriteHeader(http.StatusNotFound)

				m := &ApiResponse{
					Message: "device not found",
				}

				err := json.NewEncoder(w).Encode(m)
				if err != nil {
					log.Println(err)
				}
				return
			}

			err := json.NewEncoder(w).Encode(d)
			if err != nil {
				log.Println(err)
			}
		},
		Methods: []string{http.MethodGet},
	},
	{
		Path: "/devices/{deviceId}/act/{actuatorName}",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			deviceId := vars["deviceId"]
			actuatorName := vars["actuatorName"]

			log.Printf("Requested device %s actuation for %s\n", deviceId, actuatorName)

			d := transport.GetDeviceByID(deviceId)
			if d == nil {
				// Not found
				w.WriteHeader(http.StatusNotFound)

				m := &ApiResponse{
					Message: "device not found",
				}

				err := json.NewEncoder(w).Encode(m)
				if err != nil {
					log.Println(err)
				}
				return
			}

			err := d.TriggerActuator(actuatorName)

			data := &ApiResponse{
				Message: "Done",
			}
			if err != nil {
				data.Message = err.Error()
			}

			body, _ := json.Marshal(data)

			_, err = fmt.Fprintf(w, string(body))
			if err != nil {
				log.Println(err)
			}
		},
		Methods: []string{http.MethodPost},
	},
}

func RunServer(t *BLETransport) {
	r := mux.NewRouter()

	transport = t

	// Register endpoints
	for _, endpoint := range endpoints {
		r.HandleFunc(endpoint.Path, endpoint.Handler).
			Methods(endpoint.Methods...)
	}

	// Start server
	port := os.Getenv("GIO_FOG_NODE_SERVER_PORT")
	if port == "" {
		port = serverDefaultPort
	}
	port = fmt.Sprintf(":%s", port)

	log.Println(http.ListenAndServe(port, r))
}
