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
	"time"
)

// A GiòDevice represents a device handled by the Giò Plants platform
type GioDevice struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
	Mac  string `json:"mac"`
	Room string `json:"room"`
}

// A Reading represents a value produced by a device
type Reading struct {
	ID                string `json:"id,omitempty"`
	Name              string `json:"name"`
	Value             string `json:"value"`
	Unit              string `json:"unit"`
	CreationTimestamp string `json:"creation_timestamp"`
}

func NewReading(name string, value string, unit string) *Reading {
	return &Reading{
		Name:              name,
		Value:             value,
		Unit:              unit,
		CreationTimestamp: time.Now().UTC().String(),
	}
}

func (r Reading) String() string {
	return fmt.Sprintf("<Reading %s, %s, %s, %s, %s>", r.ID, r.Name, r.Value, r.Unit, r.CreationTimestamp)
}

// A Room is a virtual place that may contains devices
type Room struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// An ActionData stores information about an action
type ActionData struct {
	Value int `json:"value"`
}
