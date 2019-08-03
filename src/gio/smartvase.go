package gio

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/paypal/gatt"
)

const MICROBIT_NAME = "bbc micro:bit"

var light_service_id = gatt.MustParseUUID("02751625523e493b8f941765effa1b20")
var temperature_service_id = gatt.MustParseUUID("e95d6100251d470aa062fa1922dfa9a8")
var moisture_service_id = gatt.MustParseUUID("73cd5e04d32c4345a543487435c70c48")
var watering_service_id = gatt.MustParseUUID("ce9eafe4c44341db9cb581e567f3ba93")

var services = []gatt.UUID{light_service_id, temperature_service_id, moisture_service_id, watering_service_id}

var light_char_id = gatt.MustParseUUID("02759250523e493b8f941765effa1b20")
var temperature_char_id = gatt.MustParseUUID("e95d9250251d470aa062fa1922dfa9a8")
var moisture_char_id = gatt.MustParseUUID("73cd7350d32c4345a543487435c70c48")
var watering_char_id = gatt.MustParseUUID("ce9e7625c44341db9cb581e567f3ba93")

var characteristics = []gatt.UUID{light_char_id, temperature_char_id, moisture_char_id, watering_char_id}

type SmartVase struct {
	p            *gatt.Peripheral
	wateringChan chan bool
}

func (sv *SmartVase) Peripheral() *gatt.Peripheral {
	return sv.p
}

func (sv *SmartVase) TriggerWatering() {
	sv.wateringChan <- true
}

func (sv *SmartVase) String() string {
	return fmt.Sprintf("I am SmartVase %s", sv.p)
}

func (sv *SmartVase) OnPeripheralConnected(p gatt.Peripheral, stopChan chan bool) error {
	fmt.Println("SmartVase OnPeripheralConnected called")

	id := p.ID()

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
						name = "light"
					case moisture_char_id.String():
						name = "moisture"
					case temperature_char_id.String():
						name = "temperature"
					case watering_char_id.String():
						name = "watering"
					}

					fmt.Printf("%s - notified: % X | %s\n", p.Name(), b, name)

					// Send data to ms
					go func() {
						fmt.Println("Sending data to DeviceService")
						service, _ := NewDeviceService()

						r := ReadingData{Name: name, Value: string(b), Unit: ""}
						err := service.SendData(id, &r)
						if err != nil {
							fmt.Println(err.Error())
						}
					}()
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

func isMicrobit(p gatt.Peripheral, a *gatt.Advertisement) bool {
	name := strings.ToLower(p.Name())
	localname := strings.ToLower(a.LocalName)
	return (strings.Contains(name, MICROBIT_NAME) || strings.Contains(localname, MICROBIT_NAME))
}

func IsSmartVase(p gatt.Peripheral, a *gatt.Advertisement) bool {
	return isMicrobit(p, a)
}

func Create(p gatt.Peripheral) *SmartVase {
	return &SmartVase{&p, make(chan bool)}
}
