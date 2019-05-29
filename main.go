// +build

package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/paypal/gatt"
	"github.com/paypal/gatt/examples/option"
)

const MICROBIT_NAME = "bbc micro:bit"


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

var done = make(chan struct{})

var connectedPeripherals = make(map[string]bool)


func onStateChanged(d gatt.Device, s gatt.State) {
	fmt.Println("State:", s)
	switch s {
	case gatt.StatePoweredOn:
		fmt.Println("Scanning...")
		d.Scan([]gatt.UUID{}, false)
		return
	default:
		d.StopScanning()
	}
}

func isMicrobit(p gatt.Peripheral, a *gatt.Advertisement) bool {
	name := strings.ToLower(p.Name())
	localname := strings.ToLower(a.LocalName)
	return (strings.Contains(name, MICROBIT_NAME) || strings.Contains(localname, MICROBIT_NAME))
}

func isAlreadyConnected(p gatt.Peripheral) bool {
	elem, ok := connectedPeripherals[p.ID()]
	return ok && elem
}

func canConnect(p gatt.Peripheral, a *gatt.Advertisement) bool {
	return !isAlreadyConnected(p) && isMicrobit(p, a)
}

func onPeriphDiscovered(p gatt.Peripheral, a *gatt.Advertisement, rssi int) {
	if !canConnect(p, a) {
		fmt.Printf("Skipping ID:%s, NAME:(%s)\n", p.ID(), p.Name())
		return
	}

	fmt.Printf("\nPeripheral ID:%s, NAME:(%s)\n", p.ID(), p.Name())
	fmt.Println("")

	p.Device().Connect(p)
}

func onPeriphConnected(p gatt.Peripheral, err error) {
	connectedPeripherals[p.ID()] = true
	
	fmt.Println("Connected")
	defer p.Device().CancelConnection(p)

	if err := p.SetMTU(500); err != nil {
		fmt.Printf("Failed to set MTU, err: %s\n", err)
	}

	// Discovery services
	ss, err := p.DiscoverServices(services)
	if err != nil {
		fmt.Printf("Failed to discover services, err: %s\n", err)
		return
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
			
			//if c.UUID().String() == watering_char_id {
				//b, err := p.WriteCharacteristic(c, b, false)
				
				//if err != nil {
					//fmt.Printf("Sleeping for writing watering")
					//time.Sleep(5 * time.Second)
				//}
			//}

			// Subscribe the characteristic, if possible.
			if (c.Properties() & (gatt.CharNotify | gatt.CharIndicate)) != 0 {
				f := func(c *gatt.Characteristic, b []byte, err error) {
					name := c.UUID().String()
					if name == light_char_id.String() {
						name = "light char"
					} else if name == moisture_char_id.String() {
						name = "moisture char"
					} else if name == watering_char_id.String() {
						name = "watering char"
					} else if name == temperature_char_id.String() {
						name = "temp_char"
					}
					
					fmt.Printf("notified: % X | %s\n", b, name)
				}
				if err := p.SetNotifyValue(c, f); err != nil {
					fmt.Printf("Failed to subscribe characteristic, err: %s\n", err)
					continue
				}
			}

		}
		fmt.Println()
	}

	time.Sleep(5 * time.Second)
}

func onPeriphDisconnected(p gatt.Peripheral, err error) {
	fmt.Println("Disconnected")
	connectedPeripherals[p.ID()] = false
}

func main() {
	d, err := gatt.NewDevice(option.DefaultClientOptions...)
	if err != nil {
		log.Fatalf("Failed to open device, err: %s\n", err)
		return
	}

	// Register handlers.
	d.Handle(
		gatt.PeripheralDiscovered(onPeriphDiscovered),
		gatt.PeripheralConnected(onPeriphConnected),
		gatt.PeripheralDisconnected(onPeriphDisconnected),
	)

	d.Init(onStateChanged)
	
	go func() {
		// terminate after some time
		time.Sleep(20 * time.Second)
		close(done)
	}()
	
	<-done
	fmt.Println("Done")
}
