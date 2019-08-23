package gio

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

type DeviceService interface {
	GetDevices() ([]GioDevice, error)

	GetReadings(deviceId int) ([]Reading, error)
}
