package main

import (
	"fmt"
	"sync"
	"os"
	"os/signal"
	"syscall"
	"time"
	"strings"
	"errors"
	
	"github.com/paypal/gatt"
	"github.com/paypal/gatt/examples/option"
)

const MICROBIT_NAME = "bbc micro:bit"
const SCANNER_PERIOD = 10 * time.Second

var light_service_id = gatt.MustParseUUID("02751625523e493b8f941765effa1b20")
var temperature_service_id = gatt.MustParseUUID("e95d6100251d470aa062fa1922dfa9a8")
var moisture_service_id = gatt.MustParseUUID("73cd5e04d32c4345a543487435c70c48")
var watering_service_id = gatt.MustParseUUID("ce9eafe4c44341db9cb581e567f3ba93")

var services = []gatt.UUID {light_service_id, temperature_service_id, moisture_service_id, watering_service_id}

var light_char_id = gatt.MustParseUUID("02759250523e493b8f941765effa1b20")
var temperature_char_id = gatt.MustParseUUID("e95d9250251d470aa062fa1922dfa9a8")
var moisture_char_id = gatt.MustParseUUID("73cd7350d32c4345a543487435c70c48")
var watering_char_id = gatt.MustParseUUID("ce9e7625c44341db9cb581e567f3ba93")

var characteristics = []gatt.UUID {light_char_id, temperature_char_id, moisture_char_id, watering_char_id}

var stopChan = make(chan os.Signal, 1)

type BLEDevice interface {
	Peripheral() *gatt.Peripheral
	OnPeripheralConnected(p gatt.Peripheral) error
	OnPeripheralDisconnected(p gatt.Peripheral) error
}

type SmartVase struct {
	p *gatt.Peripheral
	wateringChan chan bool
}

func (sv *SmartVase) Peripheral() *gatt.Peripheral {
	return sv.p
} 

func (sv *SmartVase) String() string {
	return fmt.Sprintf("I am SmartVase %s", sv.p) 
}

func isMicrobit(p gatt.Peripheral, a *gatt.Advertisement) bool {
	name := strings.ToLower(p.Name())
	localname := strings.ToLower(a.LocalName)
	return (strings.Contains(name, MICROBIT_NAME) || strings.Contains(localname, MICROBIT_NAME))
}

func (sv *SmartVase) OnPeripheralConnected(p gatt.Peripheral) error {
	fmt.Println("SmartVase OnPeripheralConnected called")

	if err := p.SetMTU(500); err != nil {
		return errors.New(fmt.Sprintf("Failed to set MTU, err: %s\n", err))
	}

	// Discovery services
	ss, err := p.DiscoverServices(services)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to discover services, err: %s\n", err))
	}

	for _, s := range ss {
		// Discovery characteristics
		cs, err := p.DiscoverCharacteristics(characteristics, s)
		if err != nil {
			fmt.Printf("Failed to discover characteristics, err: %s\n", err)
			continue
		}

		for _, c := range cs {
			// Discovery descriptors
			_, err := p.DiscoverDescriptors(nil, c)
			if err != nil {
				fmt.Printf("Failed to discover descriptors, err: %s\n", err)
				continue
			}
			
			if c.UUID().Equal(watering_char_id) {
				go func() {
					for {
						select {
							case <-sv.wateringChan:
								if err := p.WriteCharacteristic(c, []byte{0x74}, true); err != nil {
									fmt.Printf("Failed to write on watering characteristic: %s\n", err)
								}
								fmt.Println("Written on watering characteristic")
								time.Sleep(1 * time.Second)
							case <-stopChan:
								return
						}
					}
				}()
			}

			// Subscribe the characteristic, if possible.
			if (c.Properties() & (gatt.CharNotify | gatt.CharIndicate)) != 0 {
				f := func(c *gatt.Characteristic, b []byte, err error) {
					name := c.UUID().String()
					switch name {
						case light_char_id.String():
							name = "light char"
						case moisture_char_id.String():
							name = "moisture char"
						case temperature_char_id.String():
							name = "temp_char"
						case watering_char_id.String():
							name = "watering char"
					}
					
					fmt.Printf("%s - notified: % X | %s\n", p.Name(), b, name)
				}
				if err := p.SetNotifyValue(c, f); err != nil {
					fmt.Printf("Failed to subscribe characteristic, err: %s\n", err)
					continue
				}
			}

		}
		fmt.Println()
	}
		
	<-stopChan
	
	return nil
}

func (sv *SmartVase) OnPeripheralDisconnected(p gatt.Peripheral) error {
	fmt.Println("SmartVase OnPeripheralDisconnected called")
	return nil
}

func getSmartVase(p gatt.Peripheral, a *gatt.Advertisement) (BLEDevice, error) {
	if isMicrobit(p, a) {
		return &SmartVase{&p, make(chan bool)}, nil
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

type Transport interface {
	Start(quit chan bool) error
}

type BLETransport struct {}

func (tr *BLETransport) Start(quit chan bool) error {
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
				device.OnPeripheralConnected(p)
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

func CreateBLE() *BLETransport {
	return &BLETransport{}
}

func runTransport(t Transport, wg *sync.WaitGroup) {
	defer wg.Done()
	
	quit := make(chan bool)
	err := t.Start(quit)
	if err != nil {
		fmt.Printf("Failed starting Transport, err: %s\n", err)
	}
	
	<-quit
}

func main() {
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
	
	var ble Transport
	ble = CreateBLE()
	
	transports := []Transport {
		ble,
	}
	
	var transWG sync.WaitGroup
	transWG.Add(len(transports))
	
	for _, t := range transports {
		go runTransport(t, &transWG)
	}
	
	transWG.Wait()
	
	fmt.Println("Done")
}
