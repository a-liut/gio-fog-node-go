package gio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gio-fog-node/pkg/config"
	"log"
	"net/http"
	"net/url"
)

type GioDevice struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
	Mac  string `json:"mac"`
	Room string `json:"room"`
}

type Reading struct {
	ID    string      `json:"id,omitempty"`
	Name  string      `json:"name"`
	Value interface{} `json:"value"` // It can contains any value
	Unit  string      `json:"unit"`
}

type Room struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type DeviceService struct {
	url *url.URL
}

func (ds *DeviceService) register(id string, roomName string) (*GioDevice, error) {

	// Create the room
	roomData := Room{
		Name: roomName,
	}

	roomBody, _ := json.Marshal(roomData)

	roomUrl := fmt.Sprintf("%s/rooms", ds.url)
	res, err := http.Post(roomUrl, "application/json", bytes.NewBuffer(roomBody))
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("cannot perform the requested operation: (%d) %s", res.StatusCode, res.Status)
	}

	var room Room
	if err := json.NewDecoder(res.Body).Decode(&room); err != nil {
		return nil, err
	}

	// Register the Device
	deviceData := GioDevice{
		Name: "device" + id,
		Mac:  id,
	}

	deviceBody, _ := json.Marshal(deviceData)

	devicesUrl := fmt.Sprintf("%s/rooms/%s/devices", ds.url, room.ID)
	res, err = http.Post(devicesUrl, "application/json", bytes.NewBuffer(deviceBody))
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("cannot perform the requested operation: (%d) %s", res.StatusCode, res.Status)
	}

	// Take the id from the response
	var device GioDevice
	_ = json.NewDecoder(res.Body).Decode(&device)

	return &device, nil
}

func (ds *DeviceService) SendData(device *GioDevice, reading *Reading) error {
	body, err := json.Marshal(reading)
	if err != nil {
		return err
	}

	readingsUrl := fmt.Sprintf("%s/rooms/%s/devices/%s/readings", ds.url, device.Room, device.ID)
	res, err := http.Post(readingsUrl, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		return fmt.Errorf("cannot perform the requested operation: (%d) %s", res.StatusCode, res.Status)
	}

	return nil
}

var instance *DeviceService = nil

func NewDeviceService(serviceConfig *config.DeviceServiceConfig) (*DeviceService, error) {
	if instance == nil {
		u := fmt.Sprintf("http://%s:%d", serviceConfig.Host, serviceConfig.Port)
		log.Printf("DeviceService URL: %s\n", u)

		serviceUrl, err := url.Parse(u)
		if err != nil {
			return nil, err
		}
		instance = &DeviceService{serviceUrl}
	}

	return instance, nil
}
