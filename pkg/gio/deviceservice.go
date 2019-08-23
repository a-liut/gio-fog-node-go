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
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Mac  string `json:"mac"`
	Room string `json:"room"`
}

type Reading struct {
	ID        int         `json:"id"`
	Timestamp string      `json:"timestamp"`
	Name      string      `json:"name"`
	Value     interface{} `json:"value"` // It can contains any value
	Unit      string      `json:"unit"`
	Device    GioDevice   `json:"device"`
	DeviceID  int         `json:"device_id"`
}

type ReadingData struct {
	Name  string `json:"name"`
	Value string `json:"value"` // It can contains any value
	Unit  string `json:"unit"`
}
type DeviceService struct {
	url *url.URL
}

func (ds *DeviceService) register(id string, roomId string) (string, error) {
	body := []byte(fmt.Sprintf(`{
		"mac": "%s",
		"name": "%s",
		"room_id": %s
	}`, id, "device"+id, roomId))

	res, err := http.Post(fmt.Sprintf("%s/devices", ds.url), "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	if res.StatusCode != 200 {
		return "", fmt.Errorf("cannot perform the requested operation: (%d) %s", res.StatusCode, res.Status)
	}

	// Take the id from the response
	var resBody struct {
		Id   string
		Mac  string
		Name string
		Room int
	}

	_ = json.NewDecoder(res.Body).Decode(&resBody)

	return resBody.Id, nil
}

func (ds *DeviceService) SendData(id string, reading *ReadingData) error {
	body, err := json.Marshal(reading)
	if err != nil {
		return err
	}

	res, err := http.Post(fmt.Sprintf("%s/devices/%s/readings", ds.url, id), "application/json", bytes.NewBuffer(body))
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
