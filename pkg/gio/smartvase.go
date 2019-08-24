package gio

import (
	"fmt"
	"strings"
	"time"

	"github.com/paypal/gatt"
)

const (
	microbitName = "bbc micro:bit"
	roomName     = "default"
)

var lightServiceId = gatt.MustParseUUID("02751625523e493b8f941765effa1b20")
var temperatureServiceId = gatt.MustParseUUID("e95d6100251d470aa062fa1922dfa9a8")
var moistureServiceId = gatt.MustParseUUID("73cd5e04d32c4345a543487435c70c48")
var wateringServiceId = gatt.MustParseUUID("ce9eafe4c44341db9cb581e567f3ba93")

var services = []gatt.UUID{lightServiceId, temperatureServiceId, moistureServiceId, wateringServiceId}

var lightCharId = gatt.MustParseUUID("02759250523e493b8f941765effa1b20")
var temperatureCharId = gatt.MustParseUUID("e95d9250251d470aa062fa1922dfa9a8")
var moistureCharId = gatt.MustParseUUID("73cd7350d32c4345a543487435c70c48")
var wateringCharId = gatt.MustParseUUID("ce9e7625c44341db9cb581e567f3ba93")

var characteristics = []gatt.UUID{lightCharId, temperatureCharId, moistureCharId, wateringCharId}

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

	registered := false

	service, _ := NewDeviceService(nil)

	var device *GioDevice
	go func() {
		var err error
		for !registered {

			select {
			case <-stopChan:
				fmt.Println("Stop trying to register device")
			default:
				device, err = service.register(p.ID(), roomName)
				if err == nil {
					registered = true

					fmt.Printf("Device %s registered with id: %s!", device.Name, device.ID)
				} else {
					fmt.Printf("WARNING: Cannot register the device to the DeviceService: %s\n", err)

					time.Sleep(5 * time.Second)
				}
			}
		}
	}()

	if err := p.SetMTU(500); err != nil {
		return fmt.Errorf("Failed to set MTU, err: %s\n", err)
	}

	// Discovery services
	ss, err := p.DiscoverServices(services)
	if err != nil {
		return fmt.Errorf("Failed to discover services, err: %s\n", err)
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

			if c.UUID().Equal(wateringCharId) {
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
					case lightCharId.String():
						name = "light char"
					case moistureCharId.String():
						name = "moisture char"
					case temperatureCharId.String():
						name = "temp_char"
					case wateringCharId.String():
						name = "watering char"
					}

					fmt.Printf("%s - notified: % X | %s\n", p.Name(), b, name)

					// Send data to ms
					if registered {
						go func() {
							fmt.Println("Sending data to DeviceService")

							r := Reading{
								Name:  name,
								Value: string(b),
								Unit:  "",
							}
							err := service.SendData(device, &r)
							if err != nil {
								fmt.Println(err.Error())
							} else {
								fmt.Println("Send success!")
							}
						}()
					} else {
						fmt.Println("Skipping send data: Not registered")
					}
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
	return strings.Contains(name, microbitName) || strings.Contains(localname, microbitName)
}

func IsSmartVase(p gatt.Peripheral, a *gatt.Advertisement) bool {
	return isMicrobit(p, a)
}

func Create(p gatt.Peripheral) *SmartVase {
	return &SmartVase{&p, make(chan bool)}
}
