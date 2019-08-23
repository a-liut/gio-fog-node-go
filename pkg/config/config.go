package config

type Config struct {
	DeviceServiceConfig *DeviceServiceConfig `json:"device_service"`
}

type DeviceServiceConfig struct {
	Host string `json:"host"`
	Port int32  `json:"port"`
}
