package gio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/paypal/gatt"
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

type CallbackData struct {
	Url string `json:"url"`
}

type CallbackResponseData struct {
	PeripheralID string  `json:"peripheral_id"`
	Reading      Reading `json:"reading"`
}

// Transport to be used
var transport *BLETransport
var endpoints = []Endpoint{
	{
		// Register a new callback for providing data
		Path:    "/callbacks",
		Methods: []string{http.MethodPost},
		Handler: func(w http.ResponseWriter, r *http.Request) {

			var data CallbackData
			err := json.NewDecoder(r.Body).Decode(&data)
			if err != nil {
				// Bad Request
				w.WriteHeader(http.StatusBadRequest)

				m := &ApiResponse{
					Message: "invalid data",
				}

				err := json.NewEncoder(w).Encode(m)
				if err != nil {
					log.Println(err)
				}
				return
			}

			callbackUUID := uuid.New().String()

			_ = transport.AddCallback(callbackUUID, func(peripheral gatt.Peripheral, reading Reading) {

				d := CallbackResponseData{
					PeripheralID: peripheral.ID(),
					Reading:      reading,
				}

				body, err := json.Marshal(d)
				if err != nil {
					log.Printf("error encoding reading data: %s", err)
					return
				}

				resp, err := http.Post(data.Url, "application/json", bytes.NewBuffer(body))
				if err != nil {
					log.Printf("error calling callback: %s", err)
					return
				}

				if resp.StatusCode != http.StatusOK {
					log.Printf("Callback result unsuccessful: %d\n", resp.StatusCode)

					// Remove unsuccessful callback
					_ = transport.RemoveCallback(callbackUUID)
					return
				}

				log.Printf("Callback at %s called successfully", data.Url)

				m := ApiResponse{
					Message: callbackUUID,
				}

				// Send back the UUID
				w.WriteHeader(http.StatusOK)
				err = json.NewEncoder(w).Encode(m)
				if err != nil {
					log.Println(err)
				}
			})
		},
	},
	{
		// Removes a callback
		Path:    "/callbacks/{callbackUuid}",
		Methods: []string{http.MethodDelete},
		Handler: func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			callbackUuid := vars["callbackUuid"]

			_ = transport.RemoveCallback(callbackUuid)

			w.WriteHeader(http.StatusOK)
		},
	},
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
		Path: "/devices/{deviceId}/actions/{actionName}",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			vars := mux.Vars(r)
			deviceId := vars["deviceId"]
			actionName := vars["actionName"]

			log.Printf("Requested device %s action for %s\n", deviceId, actionName)

			d := transport.GetDeviceByID(deviceId)
			if d == nil {
				// Not found
				w.WriteHeader(http.StatusNotFound)

				m := &ApiResponse{
					Message: "device not found",
				}

				w.WriteHeader(http.StatusNotFound)
				err := json.NewEncoder(w).Encode(m)
				if err != nil {
					log.Println(err)
				}
				return
			}

			err := d.TriggerAction(actionName)

			data := &ApiResponse{
				Message: "Done",
			}
			if err != nil {
				w.WriteHeader(http.StatusBadRequest) // action not recognised
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
