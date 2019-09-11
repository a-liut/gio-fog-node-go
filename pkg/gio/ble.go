package gio

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
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
	UUID           gatt.UUID               `json:"uuid"`
	Name           string                  `json:"name"`
	GetReading     func(b []byte) *Reading `json:"-"`
	Characteristic *gatt.Characteristic    `json:"-"`
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

type Callback func(p gatt.Peripheral, reading Reading) error

type CallbackMeta struct {
	ID  string
	fun Callback
}

type BLETransport struct {
	connectedPeripherals map[string]BLEConnection
	peripheralsMutex     *sync.Mutex

	callbacks      map[string]CallbackMeta
	callbacksMutex *sync.Mutex
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
	tr.peripheralsMutex.Lock()
	defer tr.peripheralsMutex.Unlock()

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
	tr.peripheralsMutex.Lock()
	defer tr.peripheralsMutex.Unlock()

	delete(tr.connectedPeripherals, p.ID())
}

func (tr *BLETransport) getDeviceConnection(p gatt.Peripheral) *BLEConnection {
	tr.peripheralsMutex.Lock()
	defer tr.peripheralsMutex.Unlock()

	d, _ := tr.connectedPeripherals[p.ID()]
	return &d
}

func getBLEDevice(p gatt.Peripheral, a *gatt.Advertisement) (BLEDevice, error) {
	if IsEnabledDevice(p, a) {
		return NewGenericBLEDevice(p), nil
	}

	return nil, fmt.Errorf("not a GenericBLEDevice")
}

func newDevice(p gatt.Peripheral, a *gatt.Advertisement) (BLEDevice, error) {
	device, err := getBLEDevice(p, a)
	if err == nil {
		return device, nil
	}

	return nil, fmt.Errorf("device not recognised: %s", err)
}

func CreateBLETransport() *BLETransport {
	return &BLETransport{
		connectedPeripherals: make(map[string]BLEConnection),
		peripheralsMutex:     &sync.Mutex{},
		callbacks:            make(map[string]CallbackMeta),
		callbacksMutex:       &sync.Mutex{},
	}
}

// Returns all connected BLE devices
func (tr *BLETransport) GetDevices() []BLEDevice {
	tr.peripheralsMutex.Lock()
	defer tr.peripheralsMutex.Unlock()

	res := make([]BLEDevice, len(tr.connectedPeripherals))

	i := 0
	for _, d := range tr.connectedPeripherals {
		res[i] = d.Device
		i++
	}

	return res
}

// Returns a BLE device with a specific ID
func (tr *BLETransport) GetDeviceByID(id string) BLEDevice {
	tr.peripheralsMutex.Lock()
	defer tr.peripheralsMutex.Unlock()

	d, _ := tr.connectedPeripherals[id]
	return d.Device
}

// Calls each registered callback when a new reading is produced. If a callback reports an error,
// the callback is removed.
func (tr *BLETransport) OnReadingProduced(peripheral gatt.Peripheral, r Reading) {
	tr.callbacksMutex.Lock()
	defer tr.callbacksMutex.Unlock()

	if len(tr.callbacks) == 0 {
		fmt.Println("WARNING: No callbacks to call!")
	}

	// Call registered callbacks
	toRemove := make([]string, 0)
	for url, meta := range tr.callbacks {
		if err := meta.fun(peripheral, r); err != nil {
			toRemove = append(toRemove, url)
		}
	}

	for _, url := range toRemove {
		log.Printf("Removing callback %s due to errors", url)
		delete(tr.callbacks, url)
	}
}

// Adds a new callback. Returns the UUID of the callback or an error
func (tr *BLETransport) AddCallback(url string, fun Callback) (string, error) {
	tr.callbacksMutex.Lock()
	defer tr.callbacksMutex.Unlock()

	if _, exists := tr.callbacks[url]; exists {
		return "", fmt.Errorf("%s already registered", url)
	}

	id := uuid.New().String()

	tr.callbacks[url] = CallbackMeta{
		ID:  id,
		fun: fun,
	}

	return id, nil
}

// Removes a callback identified by the ID
func (tr *BLETransport) RemoveCallback(id string) error {
	tr.callbacksMutex.Lock()
	defer tr.callbacksMutex.Unlock()
	delete(tr.callbacks, id)

	return nil
}

// Returns the UUID associated to url, otherwise it returns the empty string
func (tr *BLETransport) GetCallbackUUID(url string) string {
	tr.callbacksMutex.Lock()
	defer tr.callbacksMutex.Unlock()

	if meta, exists := tr.callbacks[url]; exists {
		return meta.ID
	}

	return ""
}
