package gio

import (
	"os"
	"errors"
	"sync"
	"fmt"
)

type Transport interface {
	Start(quit chan bool, stopChan chan os.Signal) error
}

type TransportRunner interface {
	Add(t Transport)
	Run() error
	Stop() error
}

type DefaultTransportRunner struct {
	transports []Transport
	isRunning bool
	stopChan chan os.Signal
}

func (sv* DefaultTransportRunner) Add(t Transport) {
	fmt.Printf("Transport added: %s", t)
	sv.transports = append(sv.transports, t)
}

func (sv* DefaultTransportRunner) runTransport(t Transport, wg *sync.WaitGroup) {
	defer wg.Done()
	
	quit := make(chan bool)
	err := t.Start(quit, sv.stopChan)
	if err != nil {
		fmt.Printf("Failed starting Transport, err: %s\n", err)
	}
	
	<-quit
}

func (sv* DefaultTransportRunner) Run() error {
	if sv.isRunning {
		return errors.New("Already running")
	}
	
	fmt.Println("Runner started")
	
	var transWG sync.WaitGroup
	transWG.Add(len(sv.transports))
		
	for _, t := range sv.transports {
		go sv.runTransport(t, &transWG)
	}
		
	transWG.Wait()
		
	fmt.Println("Runner stopped")
	
	return nil
}

func (sv* DefaultTransportRunner) Stop() error {
	close(sv.stopChan)
	return nil
}

func NewDefaultTransportRunner(stopChan chan os.Signal) TransportRunner {
	return &DefaultTransportRunner{make([]Transport, 0, 1), false, stopChan}
}
