/*
 * Fog Node
 *
 * A tool for connecting devices to the Giò Plants platform.
 *
 * API version: 1.0.0
 * Contact: andrea.liut@gmail.com
 */
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

// An Action represents an trigger request for an action
type Action struct {
	Name       string
	ActionData ActionData
}

// A GenericBLEDevice represents a connected BLE device
type GenericBLEDevice struct {
	p              *gatt.Peripheral
	actionChannels map[string]chan Action

	Services        []BLEService
	Characteristics []BLECharacteristic
}

func (sv *GenericBLEDevice) Peripheral() *gatt.Peripheral {
	return sv.p
}

// Handles the connection process of a peripheral
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

			sv.actionChannels[c.UUID().String()] = make(chan Action, 1)

			// Register action listener only if characteristic is writable
			if (c.Properties() & (gatt.CharWrite | gatt.CharWriteNR)) != 0 {
				go func() {
					log.Printf("Start action listener for characteristic %s\n", c.UUID().String())
					for {
						select {
						case <-stopChan:
							return
						case action := <-sv.actionChannels[c.UUID().String()]:
							log.Printf("Action requested: %s. Action UUID: %s", action.Name, c.UUID().String())
							if c.UUID().String() == action.Name {
								b := encodeValue(action.ActionData.Value)

								// try write on the characteristic
								if err := p.WriteCharacteristic(c, b, true); err != nil {
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

					if r == nil {
						log.Println("Skipping data notification: No value to send")
						return
					}

					// Notify data creation
					log.Printf("Reading produced: %s", r)
					go transport.OnReadingProduced(p, *r)
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

	// close action channels
	for _, channel := range sv.actionChannels {
		close(channel)
	}

	return nil
}

func encodeValue(value int) []byte {
	return []byte{byte(value)}
}

// Handles the disconnection process of a peripheral
func (sv *GenericBLEDevice) OnPeripheralDisconnected(p gatt.Peripheral) error {
	log.Println("GenericBLEDevice OnPeripheralDisconnected called")
	return nil
}

// Returns true if the peripheral is a Microbit
func isMicrobit(p gatt.Peripheral, a *gatt.Advertisement) bool {
	name := strings.ToLower(p.Name())
	localname := strings.ToLower(a.LocalName)
	return strings.Contains(name, microbitName) || strings.Contains(localname, microbitName)
}

// Creates a new reading from data sent from a BLE Characteristic
func parseReading(c *gatt.Characteristic, b []byte) *Reading {
	return NewReading(c.UUID().String(), fmt.Sprintf("%v", b), "")
}

// Returns if the device is authorised for connection
func IsEnabledDevice(p gatt.Peripheral, a *gatt.Advertisement) bool {
	return isMicrobit(p, a)
}

func NewGenericBLEDevice(p gatt.Peripheral) *GenericBLEDevice {
	return &GenericBLEDevice{
		p:              &p,
		actionChannels: make(map[string]chan Action),
	}
}

func (sv *GenericBLEDevice) AvailableCharacteristics() []BLECharacteristic {
	return sv.Characteristics
}

// Triggers an action on a device
func (sv *GenericBLEDevice) TriggerAction(actionName string, data ActionData) error {
	log.Printf("Triggering %s\n", actionName)
	channel, exists := sv.actionChannels[actionName]
	if !exists {
		return fmt.Errorf("action %s not recognised", actionName)
	}

	channel <- Action{Name: actionName, ActionData: data}

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
