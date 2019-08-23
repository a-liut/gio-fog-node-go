package gio

import (
	"github.com/paypal/gatt"
)

type BLEDevice interface {
	Peripheral() *gatt.Peripheral
	OnPeripheralConnected(p gatt.Peripheral, stopChan chan bool) error
	OnPeripheralDisconnected(p gatt.Peripheral) error
}
