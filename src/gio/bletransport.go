package gio

import (
	"fmt"
	"errors"
	"time"
	"os"
	"sync"
	
	"github.com/paypal/gatt"
	"github.com/paypal/gatt/examples/option"
)

const SCANNER_PERIOD = 10 * time.Second

type BLETransport struct {}

func (tr *BLETransport) Start(quit chan bool, stopChan chan os.Signal) error {
	defer close(quit)
	
	fmt.Println("BLE init called")
	
	d, err := gatt.NewDevice(option.DefaultClientOptions...)
	if err != nil {
		return err
	}
	
	connectedPeripherals := make(map[string]BLEDevice)
	cpMutex := &sync.Mutex{}
	
	// Register handlers.
	d.Handle(
		gatt.PeripheralDiscovered(func (p gatt.Peripheral, a *gatt.Advertisement, rssi int) {
			device, err := getDevice(p, a)
			if err != nil {
				// fmt.Printf("ERROR: %s\n", err)
				return
			}
			
			fmt.Printf("Setting device for p: %s (%s)\n", p.ID(), p.Name())
			cpMutex.Lock()
			if _, alreadyPresent := connectedPeripherals[p.ID()]; alreadyPresent {
				fmt.Printf("Peripheral %s (%s) already connected\n", p.ID(), p.Name())
				
				cpMutex.Unlock()
				return
			}
			
			connectedPeripherals[p.ID()] = device
			cpMutex.Unlock()
				
			p.Device().Connect(p)
		}),
		gatt.PeripheralConnected(func (p gatt.Peripheral, err error) {
			fmt.Printf("BLE device connected: %s (%s)\n", p.ID(), p.Name())
			
			defer p.Device().CancelConnection(p)
			
			cpMutex.Lock()
			device, ok := connectedPeripherals[p.ID()]
			cpMutex.Unlock()
			if ok {
				fmt.Println("Calling OnPeripheralConnected...")
				device.OnPeripheralConnected(p, stopChan)
			} else {
				fmt.Printf("OnPeripheralConnected: Connected device for ID %s not found. Maybe something went wrong...\n", p.ID())
			}
		}),
		gatt.PeripheralDisconnected(func (p gatt.Peripheral, err error) {
			fmt.Printf("BLE device disconnected: %s (%s)\n", p.ID(), p.Name())
			
			cpMutex.Lock()
			device, ok := connectedPeripherals[p.ID()]
			cpMutex.Unlock()
			if ok {
				fmt.Println("Calling OnPeripheralDisconnected...")
				device.OnPeripheralDisconnected(p)
			} else {
				fmt.Printf("PeripheralDisconnected: Connected device for ID %s not found. Maybe something went wrong...\n", p.ID())
			}
			
			cpMutex.Lock()
			delete(connectedPeripherals, p.ID())
			cpMutex.Unlock()
		}),
	)

	d.Init(func (d gatt.Device, s gatt.State) {
		switch s {
		case gatt.StatePoweredOn:
			go func() {
				ticker := time.NewTicker(SCANNER_PERIOD)
				defer ticker.Stop()
				
				fmt.Println("Scanning...")
				d.Scan([]gatt.UUID{}, false)
				
				for {
					select {
					case <-ticker.C:
						d.StopScanning()
						
						fmt.Println("Scanning...")
						d.Scan([]gatt.UUID{}, false)
					case <-quit:
						fmt.Println("Stop scanning...")
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

func CreateBLETransport() *BLETransport {
	return &BLETransport{}
}

func getSmartVase(p gatt.Peripheral, a *gatt.Advertisement) (BLEDevice, error) {
	if IsSmartVase(p, a) {
		return Create(p), nil
	}
	
	return nil, errors.New("Not a SmartVase")
}

func getDevice(p gatt.Peripheral, a *gatt.Advertisement) (BLEDevice, error) {
	device, err := getSmartVase(p, a)
	if err == nil {
		return device, nil
	}
	
	return nil, errors.New(fmt.Sprintf("Device not recognised: %s", err))
}
