package gio

import "fmt"

type GioDevice struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
	Mac  string `json:"mac"`
	Room string `json:"room"`
}

type Reading struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name"`
	Value string `json:"value"`
	Unit  string `json:"unit"`
}

func NewReading(name string, value string, unit string) *Reading {
	return &Reading{Name: name, Value: value, Unit: unit}
}

func (r Reading) String() string {
	return fmt.Sprintf("<Reading %s, %s, %s, %s>", r.ID, r.Name, r.Value, r.Unit)
}

type Room struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
