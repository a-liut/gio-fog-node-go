package gio

type DeviceService interface {
	
	GetDevices() ([]GioDevice, error)
	
	GetReadings(deviceId int) ([]Reading, error)
	
}
