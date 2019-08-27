package gio

import (
	"encoding/json"
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

	AvailableCharacteristics() []BLECharacteristic
	TriggerAction(actuatorName string) error
}

type BLEService struct {
	UUID gatt.UUID `json:"uuid"`
	Name string    `json:"name"`
}

func (bles BLEService) String() string {
	return bles.UUID.String()
}

type BLECharacteristic struct {
	UUID       gatt.UUID               `json:"uuid"`
	Name       string                  `json:"name"`
	GetReading func(b []byte) *Reading `json:"-"`
}

func (blec *BLECharacteristic) MarshalJSON() ([]byte, error) {
	type Alias BLECharacteristic

	return json.Marshal(&struct {
		UUID string `json:"uuid"`
		Name string `json:"name"`
		*Alias
	}{
		UUID:  blec.UUID.String(),
		Name:  blec.Name,
		Alias: (*Alias)(blec),
	})
}

func (blec BLECharacteristic) String() string {
	return blec.UUID.String()
}

const (
	scannerPeriod = 10 * time.Second
)

type BLEConnection struct {
	Device            BLEDevice
	connectionChannel chan struct{}
}

func (conn *BLEConnection) Close() {
	p := *conn.Device.Peripheral()
	log.Printf("Closing connection with device %s\n", p.ID())
	close(conn.connectionChannel)
}

type BLETransport struct {
	connectedPeripherals map[string]BLEConnection
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

			conn := tr.getDeviceConnection(p)
			if conn != nil {
				log.Println("Calling OnPeripheralConnected...")
				conn.Device.OnPeripheralConnected(p, conn.connectionChannel)
			} else {
				log.Printf("OnPeripheralConnected: Connected device for ID %s not found. Maybe something went wrong...\n", p.ID())
			}
		}),
		gatt.PeripheralDisconnected(func(p gatt.Peripheral, err error) {
			log.Printf("BLE device disconnected: %s (%s)\n", p.ID(), p.Name())

			conn := tr.getDeviceConnection(p)
			if conn != nil {
				log.Println("Calling OnPeripheralDisconnected...")
				conn.Device.OnPeripheralDisconnected(p)
				conn.Close()
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

						// Close connections
						for _, conn := range tr.connectedPeripherals {
							conn.Close()
						}

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

	tr.connectedPeripherals[p.ID()] = BLEConnection{
		Device:            device,
		connectionChannel: make(chan struct{}),
	}
}

func (tr *BLETransport) removePeripheral(p gatt.Peripheral) {
	tr.cpMutex.Lock()
	defer tr.cpMutex.Unlock()

	delete(tr.connectedPeripherals, p.ID())
}

func (tr *BLETransport) getDeviceConnection(p gatt.Peripheral) *BLEConnection {
	tr.cpMutex.Lock()
	defer tr.cpMutex.Unlock()

	d, _ := tr.connectedPeripherals[p.ID()]
	return &d
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

func CreateBLETransport() *BLETransport {
	return &BLETransport{
		connectedPeripherals: make(map[string]BLEConnection),
		cpMutex:              &sync.Mutex{},
	}
}

func (tr *BLETransport) GetDevices() []BLEDevice {
	tr.cpMutex.Lock()
	defer tr.cpMutex.Unlock()

	res := make([]BLEDevice, len(tr.connectedPeripherals))

	i := 0
	for _, d := range tr.connectedPeripherals {
		res[i] = d.Device
		i++
	}

	return res
}

func (tr *BLETransport) GetDeviceByID(id string) BLEDevice {
	tr.cpMutex.Lock()
	defer tr.cpMutex.Unlock()

	d, _ := tr.connectedPeripherals[id]
	return d.Device
}
