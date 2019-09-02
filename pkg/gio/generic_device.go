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
)

type Action struct {
	Name string
}

type GenericBLEDevice struct {
	p             *gatt.Peripheral
	actionChannel chan Action

	Services        []BLEService
	Characteristics []BLECharacteristic
}

func (sv *GenericBLEDevice) Peripheral() *gatt.Peripheral {
	return sv.p
}

func (sv *GenericBLEDevice) OnPeripheralConnected(p gatt.Peripheral, stopChan chan struct{}) error {
	log.Println("GenericBLEDevice OnPeripheralConnected called")

	if err := p.SetMTU(500); err != nil {
		return fmt.Errorf("Failed to set MTU, err: %s\n", err)
	}

	// Discovery services
	ss, err := p.DiscoverServices(nil)
	if err != nil {
		return fmt.Errorf("Failed to discover services, err: %s\n", err)
	}

	for _, s := range ss {

		sv.Services = append(sv.Services, BLEService{
			UUID: s.UUID(),
			Name: s.Name(),
		})

		// Discover characteristics
		cs, err := p.DiscoverCharacteristics(nil, s)
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

			sv.Characteristics = append(sv.Characteristics, BLECharacteristic{
				UUID:       c.UUID(),
				Name:       c.Name(),
				GetReading: nil,
			})

			// Register action listener only if characteristic is writable
			if (c.Properties() & (gatt.CharWrite | gatt.CharWriteNR)) != 0 {
				go func() {
					log.Printf("Start action listener for characteristic %s\n", c.UUID())
					for {
						select {
						case <-stopChan:
							return
						case action := <-sv.actionChannel:
							if c.UUID().String() == action.Name {
								// try write on the characteristic
								if err := p.WriteCharacteristic(c, []byte{0x74}, true); err != nil {
									log.Printf("Failed to write on watering characteristic %s: %s\n", c.UUID(), err)
								}
								log.Printf("Written on characteristic %s\n", c.UUID())
								time.Sleep(1 * time.Second)
							}
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
					go func() {
						log.Println("Sending data to DeviceService")

						log.Printf("<%s, %s, %s>\n", r.Name, r.Value, r.Unit)

						transport.OnReadingProduced(p, *r)
					}()
				}

				if err := p.SetNotifyValue(c, f); err != nil {
					log.Printf("Failed to subscribe characteristic, err: %s\n", err)
					continue
				}
			}

		}

		log.Println("-----")
	}

	<-stopChan

	return nil
}

func (sv *GenericBLEDevice) OnPeripheralDisconnected(p gatt.Peripheral) error {
	log.Println("GenericBLEDevice OnPeripheralDisconnected called")
	return nil
}

func isMicrobit(p gatt.Peripheral, a *gatt.Advertisement) bool {
	name := strings.ToLower(p.Name())
	localname := strings.ToLower(a.LocalName)
	return strings.Contains(name, microbitName) || strings.Contains(localname, microbitName)
}

func parseReading(c *gatt.Characteristic, b []byte) *Reading {
	return &Reading{
		Name:  c.UUID().String(),
		Value: fmt.Sprintf("%v", b),
		Unit:  "",
	}
}

func IsEnabledDevice(p gatt.Peripheral, a *gatt.Advertisement) bool {
	return isMicrobit(p, a)
}

func Create(p gatt.Peripheral) *GenericBLEDevice {
	return &GenericBLEDevice{
		p:             &p,
		actionChannel: make(chan Action),
	}
}

func (sv *GenericBLEDevice) AvailableCharacteristics() []BLECharacteristic {
	return sv.Characteristics
}

func (sv *GenericBLEDevice) TriggerAction(actionName string) error {
	sv.actionChannel <- Action{Name: actionName}
	return nil
}

func (sv *GenericBLEDevice) MarshalJSON() ([]byte, error) {
	type Alias GenericBLEDevice

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
