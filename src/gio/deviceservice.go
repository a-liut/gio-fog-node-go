package gio

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func (ds *DeviceService) SendData(id string, reading *ReadingData) error {
	body, err := json.Marshal(reading)
	if err != nil {
		return err
	}

	resp, err := http.Post(fmt.Sprintf("%s/devices/%s/readings", ds.url, id), "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("cannot perform the requested operation: (%d) %s", resp.StatusCode, resp.Status)
	}

	return nil
}

var INSTANCE *DeviceService = nil

func NewDeviceService() (*DeviceService, error) {
	if INSTANCE == nil {
		url, err := url.Parse("http://192.168.99.100:31334")
		if err != nil {
			return nil, err
		}
		INSTANCE = &DeviceService{url}
	}

	return INSTANCE, nil
}
