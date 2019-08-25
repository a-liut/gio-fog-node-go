package gio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
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

func (ds *DeviceService) Register(id string, roomName string) (*GioDevice, error) {

	// Create the room
	roomData := Room{
		Name: roomName,
	}

	roomBody, _ := json.Marshal(roomData)

	roomUrl := fmt.Sprintf("%s/rooms", ds.url)
	roomResponse, err := http.Post(roomUrl, "application/json", bytes.NewBuffer(roomBody))

	if err != nil {
		return nil, err
	}

	defer roomResponse.Body.Close()

	if roomResponse.StatusCode != 200 {
		return nil, fmt.Errorf("cannot perform the requested operation: (%d) %s", roomResponse.StatusCode, roomResponse.Status)
	}

	var room Room
	if err := json.NewDecoder(roomResponse.Body).Decode(&room); err != nil {
		return nil, err
	}

	// Register the Device
	deviceData := GioDevice{
		Name: "device" + id,
		Mac:  id,
	}

	deviceBody, _ := json.Marshal(deviceData)

	devicesUrl := fmt.Sprintf("%s/rooms/%s/devices", ds.url, room.ID)
	deviceResponse, err := http.Post(devicesUrl, "application/json", bytes.NewBuffer(deviceBody))

	if err != nil {
		return nil, err
	}

	defer deviceResponse.Body.Close()

	if deviceResponse.StatusCode != 200 {
		return nil, fmt.Errorf("cannot perform the requested operation: (%d) %s", deviceResponse.StatusCode, deviceResponse.Status)
	}

	// Take the id from the response
	var device GioDevice
	_ = json.NewDecoder(deviceResponse.Body).Decode(&device)

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

	defer res.Body.Close()

	if res.StatusCode != 200 {
		return fmt.Errorf("cannot perform the requested operation: (%d) %s", res.StatusCode, res.Status)
	}

	return nil
}

var instance *DeviceService = nil

func NewDeviceService() (*DeviceService, error) {
	serviceHost := os.Getenv("DEVICE_SERVICE_HOST")
	servicePort := os.Getenv("DEVICE_SERVICE_PORT")

	if instance == nil {
		u := fmt.Sprintf("http://%s:%s", serviceHost, servicePort)
		log.Printf("DeviceService URL: %s\n", u)

		serviceUrl, err := url.Parse(u)
		if err != nil {
			return nil, err
		}
		instance = &DeviceService{serviceUrl}
	}

	return instance, nil
}
