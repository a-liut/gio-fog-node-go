package main

import (
	"fmt"
	"log"
	"strings"
	"time"
	"net/http"
	"os"
	"os/signal"
	"syscall"

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

var watering_chan = make(chan int)
var exit_chan = make(chan os.Signal, 1)


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
	
	var quit = make(chan struct{})

	if err := p.SetMTU(500); err != nil {
		fmt.Printf("Failed to set MTU, err: %s\n", err)
	}

	// Discovery services
	ss, err := p.DiscoverServices(services)
	if err != nil {
		fmt.Printf("Failed to discover services, err: %s\n", err)
		return
	}
	
	// readingMap := make(map[string]float32)

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
							case <-watering_chan:
								if err := p.WriteCharacteristic(c, []byte{0x74}, true); err != nil {
									fmt.Printf("Failed to write on watering characteristic: %s\n", err)
								}
								fmt.Println("Written on watering characteristic")
								time.Sleep(1 * time.Second)
							case <-quit:
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
					
					fmt.Printf("notified: % X | %s\n", b, name)
					
					//fmt.Printf("adding %s to readingMap\n", c.UUID().String())
					//v, ok := readingMap[c.UUID().String()]
					//if !ok {
						//v = 0
					//}
					//fmt.Println(readingMap)
					//readingMap[c.UUID().String()] = v * 0.8 + float32(b[8]) * 0.2
					//fmt.Println(">ok")
				}
				if err := p.SetNotifyValue(c, f); err != nil {
					fmt.Printf("Failed to subscribe characteristic, err: %s\n", err)
					continue
				}
			}

		}
		fmt.Println()
	}

	<-exit_chan
	close(quit)
	
	// Send data to MS
	//for charkey, value := range readingMap {
		//fmt.Printf("Sending %d for char %s\n", value, charkey)
	//}
}

func onPeriphDisconnected(p gatt.Peripheral, err error) {
	fmt.Println("Disconnected")
	connectedPeripherals[p.ID()] = false
}

func setupServer() {
	http.HandleFunc("/watering", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Watering request!")
		watering_chan <- 1
	})
	
	http.ListenAndServe(":3003", nil)
}

func main() {
	d, err := gatt.NewDevice(option.DefaultClientOptions...)
	if err != nil {
		log.Fatalf("Failed to open device, err: %s\n", err)
		return
	}
	
	go setupServer()

	// Register handlers.
	d.Handle(
		gatt.PeripheralDiscovered(onPeriphDiscovered),
		gatt.PeripheralConnected(onPeriphConnected),
		gatt.PeripheralDisconnected(onPeriphDisconnected),
	)

	d.Init(onStateChanged)
	
	signal.Notify(exit_chan, os.Interrupt, syscall.SIGTERM)
	go func() {
		// terminate after some interrupt
		select {
			case <-exit_chan:
				close(done)
		}
	}()
	
	<-done
	fmt.Println("Done")
}
