package gio

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/paypal/gatt"
	"github.com/paypal/gatt/examples/option"
)

type BLEDevice interface {
	Peripheral() *gatt.Peripheral
	OnPeripheralConnected(p gatt.Peripheral, stopChan chan struct{}) error
	OnPeripheralDisconnected(p gatt.Peripheral) error
}

const (
	scannerPeriod = 10 * time.Second
)

type BLETransport struct {
	connectedPeripherals map[string]BLEDevice
	cpMutex              *sync.Mutex
}

func (tr *BLETransport) Start(stopChan chan struct{}) error {
	log.Println("BLE init called")

	d, err := gatt.NewDevice(option.DefaultClientOptions...)
	if err != nil {
		return err
	}

	// Register handlers.
	d.Handle(
		gatt.PeripheralDiscovered(func(p gatt.Peripheral, a *gatt.Advertisement, rssi int) {
			device, err := newDevice(p, a)
			if err != nil {
				return
			}

			log.Printf("Setting device for p: %s (%s)\n", p.ID(), p.Name())
			tr.addPeripheral(p, device)

			p.Device().Connect(p)
		}),
		gatt.PeripheralConnected(func(p gatt.Peripheral, err error) {
			log.Printf("BLE device connected: %s (%s)\n", p.ID(), p.Name())

			defer p.Device().CancelConnection(p)

			device := tr.getDevice(p)
			if device != nil {
				log.Println("Calling OnPeripheralConnected...")
				device.OnPeripheralConnected(p, stopChan)
			} else {
				log.Printf("OnPeripheralConnected: Connected device for ID %s not found. Maybe something went wrong...\n", p.ID())
			}
		}),
		gatt.PeripheralDisconnected(func(p gatt.Peripheral, err error) {
			log.Printf("BLE device disconnected: %s (%s)\n", p.ID(), p.Name())

			device := tr.getDevice(p)
			if device != nil {
				log.Println("Calling OnPeripheralDisconnected...")
				device.OnPeripheralDisconnected(p)
			} else {
				log.Printf("PeripheralDisconnected: Connected device for ID %s not found. Maybe something went wrong...\n", p.ID())
			}

			tr.removePeripheral(p)
		}),
	)

	d.Init(func(d gatt.Device, s gatt.State) {
		switch s {
		case gatt.StatePoweredOn:
			go func() {
				ticker := time.NewTicker(scannerPeriod)
				defer ticker.Stop()

				log.Println("Scanning...")
				d.Scan([]gatt.UUID{}, false)

				for {
					select {
					case <-ticker.C:
						d.StopScanning()

						log.Println("Scanning...")
						d.Scan([]gatt.UUID{}, false)
					case <-stopChan:
						log.Println("Stop scanning...")
						return
					}
				}
			}()

			return
		default:
			d.StopScanning()
		}
	})

	<-stopChan

	return nil
}

func (tr *BLETransport) String() string {
	return "<BLETransport>"
}

func (tr *BLETransport) addPeripheral(p gatt.Peripheral, device BLEDevice) {
	tr.cpMutex.Lock()
	defer tr.cpMutex.Unlock()

	if _, alreadyPresent := tr.connectedPeripherals[p.ID()]; alreadyPresent {
		log.Printf("Peripheral %s (%s) already connected\n", p.ID(), p.Name())

		return
	}

	tr.connectedPeripherals[p.ID()] = device
}

func (tr *BLETransport) removePeripheral(p gatt.Peripheral) {
	tr.cpMutex.Lock()
	defer tr.cpMutex.Unlock()

	delete(tr.connectedPeripherals, p.ID())
}

func (tr *BLETransport) getDevice(p gatt.Peripheral) BLEDevice {
	tr.cpMutex.Lock()
	defer tr.cpMutex.Unlock()

	d, _ := tr.connectedPeripherals[p.ID()]
	return d
}

func CreateBLETransport() *BLETransport {
	return &BLETransport{
		connectedPeripherals: make(map[string]BLEDevice),
		cpMutex:              &sync.Mutex{},
	}
}

func getSmartVase(p gatt.Peripheral, a *gatt.Advertisement) (BLEDevice, error) {
	if IsSmartVase(p, a) {
		return Create(p), nil
	}

	return nil, fmt.Errorf("not a SmartVase")
}

func newDevice(p gatt.Peripheral, a *gatt.Advertisement) (BLEDevice, error) {
	device, err := getSmartVase(p, a)
	if err == nil {
		return device, nil
	}

	return nil, fmt.Errorf("device not recognised: %s", err)
}
