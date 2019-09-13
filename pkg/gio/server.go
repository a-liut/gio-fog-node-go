package gio

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
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
				code := http.StatusBadRequest
				w.WriteHeader(code)

				m := &ApiResponse{
					Code:    code,
					Message: "invalid data",
				}

				err := json.NewEncoder(w).Encode(m)
				if err != nil {
					log.Println(err)
				}
				return
			}

			callbackUUID := transport.GetCallbackUUID(data.Url)
			if callbackUUID == "" {
				log.Printf("Adding callback %s...", data.Url)
				// Add the new callback
				callbackUUID, err = transport.AddCallback(data.Url, func(peripheral gatt.Peripheral, reading Reading) error {
					d := CallbackResponseData{
						PeripheralID: peripheral.ID(),
						Reading:      reading,
					}

					body, err := json.Marshal(d)
					if err != nil {
						log.Printf("error encoding reading data: %s", err)
						return err
					}

					log.Printf("Calling callback %s\n", data.Url)
					resp, err := http.Post(data.Url, "application/json", bytes.NewBuffer(body))
					if err != nil {
						log.Printf("error calling callback: %s", err)
						return nil
					}

					if resp.StatusCode != http.StatusOK {
						log.Printf("Callback result unsuccessful: (%d) %s\n", resp.StatusCode, resp.Status)
						//return fmt.Errorf("Callback result unsuccessful: (%d) %s\n", resp.StatusCode, resp.Status)
						return nil
					}

					log.Printf("Callback %s called successfully", data.Url)

					return nil
				})
				if err != nil {
					code := http.StatusInternalServerError
					w.WriteHeader(code)

					m := &ApiResponse{
						Code:    code,
						Message: err.Error(),
					}

					err := json.NewEncoder(w).Encode(m)
					if err != nil {
						log.Println(err)
					}
					return
				}
			}

			log.Printf("Callback added %s\n", data.Url)

			m := ApiResponse{
				Message: callbackUUID,
			}

			// Send back the UUID
			w.WriteHeader(http.StatusOK)
			err = json.NewEncoder(w).Encode(m)
			if err != nil {
				log.Println(err)
			}
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

			d := transport.GetDeviceByID(deviceId)
			if d == nil {
				// Not found
				code := http.StatusNotFound
				w.WriteHeader(code)

				m := &ApiResponse{
					Code:    code,
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

			log.Printf("Requested device %s action for %s\n", actionName, deviceId)

			// Try to get data
			var data ActionData
			err := json.NewDecoder(r.Body).Decode(&data)
			if err != nil {
				data.Value = 116
			}

			d := transport.GetDeviceByID(deviceId)
			if d == nil {
				// Not found
				code := http.StatusNotFound
				w.WriteHeader(code)

				m := &ApiResponse{
					Code:    code,
					Message: "device not found",
				}

				err := json.NewEncoder(w).Encode(m)
				if err != nil {
					log.Println(err)
				}
				return
			}

			log.Printf("Device found: %s\n", d)

			err = d.TriggerAction(actionName, data)

			resp := &ApiResponse{
				Code:    http.StatusOK,
				Message: "Done",
			}
			if err != nil {
				// action not recognised
				resp.Code = http.StatusBadRequest
				resp.Message = err.Error()
			}

			log.Printf("Answer: (%d) %s", resp.Code, resp.Message)

			w.WriteHeader(resp.Code)

			err = json.NewEncoder(w).Encode(resp)
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

	log.Printf("FogNode REST interface started on port %s", port)

	port = fmt.Sprintf(":%s", port)

	log.Println(http.ListenAndServe(port, r))
}
