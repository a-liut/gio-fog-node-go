package gio

import (
	"os"

	"github.com/paypal/gatt"
)

type BLEDevice interface {
	Peripheral() *gatt.Peripheral
	OnPeripheralConnected(p gatt.Peripheral, stopChan chan os.Signal) error
	OnPeripheralDisconnected(p gatt.Peripheral) error
}
