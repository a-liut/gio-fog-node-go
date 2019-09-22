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
	"fmt"
	"github.com/paypal/gatt"
	"log"
	"sync"
)

type Transport interface {
	Start(stopChan chan struct{}) error

	OnReadingProduced(peripheral gatt.Peripheral, r Reading)
	AddCallback(id string, fun Callback) (string, error)
	RemoveCallback(id string) error
}

type TransportRunner interface {
	Add(t Transport)
	Run() error
	Stop() error
}

type DefaultTransportRunner struct {
	transports []Transport
	isRunning  bool
	transWG    sync.WaitGroup
	stopChan   chan struct{}
}

func (sv *DefaultTransportRunner) Add(t Transport) {
	sv.transports = append(sv.transports, t)
}

func (sv *DefaultTransportRunner) runTransport(t Transport, wg *sync.WaitGroup) {
	defer wg.Done()

	err := t.Start(sv.stopChan)
	if err != nil {
		log.Printf("Failed starting Transport, err: %s\n", err)
	}
}

func (sv *DefaultTransportRunner) Run() error {
	if sv.isRunning {
		return fmt.Errorf("already running")
	}

	sv.transWG.Add(len(sv.transports))

	for _, t := range sv.transports {
		go sv.runTransport(t, &sv.transWG)
	}

	return nil
}

func (sv *DefaultTransportRunner) Stop() error {
	close(sv.stopChan)
	sv.transWG.Wait()
	return nil
}

func NewDefaultTransportRunner() TransportRunner {
	return &DefaultTransportRunner{make([]Transport, 0, 1), false, sync.WaitGroup{}, make(chan struct{}, 1)}
}
