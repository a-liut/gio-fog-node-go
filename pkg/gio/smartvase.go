package gio

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/paypal/gatt"
)

const (
	microbitName = "bbc micro:bit"
	roomName     = "default"
)

var smartVaseServices = []BLEService{
	{
		UUID: gatt.MustParseUUID("02751625523e493b8f941765effa1b20"),
		Name: "light",
	},
	{
		UUID: gatt.MustParseUUID("e95d6100251d470aa062fa1922dfa9a8"),
		Name: "temperature",
	},
	{
		UUID: gatt.MustParseUUID("73cd5e04d32c4345a543487435c70c48"),
		Name: "moisture",
	},
	{
		UUID: gatt.MustParseUUID("ce9eafe4c44341db9cb581e567f3ba93"),
		Name: "watering",
	},
}

var smartVaseCharacteristics = []BLECharacteristic{
	{
		UUID: gatt.MustParseUUID("02759250523e493b8f941765effa1b20"),
		Name: "light",
		GetReading: func(b []byte) *Reading {
			return &Reading{
				Name:  "light",
				Value: fmt.Sprintf("%v", b),
				Unit:  "",
			}
		},
	},
	{
		UUID: gatt.MustParseUUID("e95d9250251d470aa062fa1922dfa9a8"),
		Name: "temperature",
		GetReading: func(b []byte) *Reading {
			v := fmt.Sprintf("%v", b)
			return &Reading{
				Name:  "temperature",
				Value: v[1 : len(v)-1],
				Unit:  "C°",
			}
		},
	},
	{
		UUID: gatt.MustParseUUID("73cd7350d32c4345a543487435c70c48"),
		Name: "moisture",
		GetReading: func(b []byte) *Reading {
			v := fmt.Sprintf("%v", b)
			return &Reading{
				Name:  "moisture",
				Value: v[1 : len(v)-1],
				Unit:  "",
			}
		},
	},
	{
		UUID: gatt.MustParseUUID("ce9e7625c44341db9cb581e567f3ba93"),
		Name: "watering",
		GetReading: func(b []byte) *Reading {
			return nil
		},
	},
}

var services []gatt.UUID
var characteristics []gatt.UUID

var wateringCharUUID *gatt.UUID

func init() {
	characteristics = make([]gatt.UUID, len(smartVaseCharacteristics))
	for i, c := range smartVaseCharacteristics {
		characteristics[i] = c.UUID

		if c.Name == "watering" {
			wateringCharUUID = &characteristics[i]
		}
	}

	services = make([]gatt.UUID, len(smartVaseServices))
	for i, s := range smartVaseServices {
		services[i] = s.UUID
	}
}

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

func (sv *SmartVase) OnPeripheralConnected(p gatt.Peripheral, stopChan chan struct{}) error {
	log.Println("SmartVase OnPeripheralConnected called")

	registered := false

	service, _ := NewDeviceService()

	var device *GioDevice
	go func() {
		var err error
		for !registered {

			select {
			case <-stopChan:
				log.Println("Stop trying to register device")
			default:
				device, err = service.Register(p.ID(), roomName)
				if err == nil {
					registered = true

					log.Printf("Device %s registered with id: %s!", device.Name, device.ID)
				} else {
					log.Printf("WARNING: Cannot register the device to the DeviceService: %s\n", err)

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
			log.Printf("Failed to discover characteristics, err: %s\n", err)
			continue
		}

		for _, c := range cs {
			// Discovery descriptors
			_, err := p.DiscoverDescriptors(nil, c)
			if err != nil {
				log.Printf("Failed to discover descriptors, err: %s\n", err)
				continue
			}

			if c.UUID().Equal(*wateringCharUUID) {
				go func() {
					for {
						select {
						case <-sv.wateringChan:
							if err := p.WriteCharacteristic(c, []byte{0x74}, true); err != nil {
								log.Printf("Failed to write on watering characteristic: %s\n", err)
							}
							log.Println("Written on watering characteristic")
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

					r := parseReading(c, b)

					log.Printf("%s - notified: %v | %s\n", p.Name(), b, c.UUID().String())

					if r == nil {
						log.Println("Skipping sending data: No value to send")
						return
					}

					// Send data to ms
					if registered {
						go func() {
							log.Println("Sending data to DeviceService")

							log.Printf("<%s, %s, %s>\n", r.Name, r.Value, r.Unit)

							err := service.SendData(device, r)
							if err != nil {
								log.Println(err.Error())
							} else {
								log.Println("Send success!")
							}
						}()
					} else {
						log.Println("Skipping sending data: Not registered")
					}
				}

				if err := p.SetNotifyValue(c, f); err != nil {
					log.Printf("Failed to subscribe characteristic, err: %s\n", err)
					continue
				}
			}

		}
		log.Println()
	}

	<-stopChan

	return nil
}

func (sv *SmartVase) OnPeripheralDisconnected(p gatt.Peripheral) error {
	log.Println("SmartVase OnPeripheralDisconnected called")
	return nil
}

func isMicrobit(p gatt.Peripheral, a *gatt.Advertisement) bool {
	name := strings.ToLower(p.Name())
	localname := strings.ToLower(a.LocalName)
	return strings.Contains(name, microbitName) || strings.Contains(localname, microbitName)
}

func parseReading(c *gatt.Characteristic, b []byte) *Reading {
	for _, char := range smartVaseCharacteristics {
		if c.UUID().Equal(char.UUID) {
			return char.GetReading(b)
		}
	}

	return nil
}

func IsSmartVase(p gatt.Peripheral, a *gatt.Advertisement) bool {
	return isMicrobit(p, a)
}

func Create(p gatt.Peripheral) *SmartVase {
	return &SmartVase{
		p:            &p,
		wateringChan: make(chan bool),
	}
}

func (sv *SmartVase) AvailableCharacteristics() []BLECharacteristic {
	return smartVaseCharacteristics
}

func (sv *SmartVase) MarshalJSON() ([]byte, error) {
	type Alias SmartVase

	return json.Marshal(&struct {
		ID              string              `json:"id"`
		Name            string              `json:"name"`
		Characteristics []BLECharacteristic `json:"characteristics"`
		*Alias
	}{
		ID:              (*sv.Peripheral()).ID(),
		Name:            (*sv.Peripheral()).Name(),
		Characteristics: sv.AvailableCharacteristics(),
		Alias:           (*Alias)(sv),
	})
}
