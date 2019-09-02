package gio

type GioDevice struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
	Mac  string `json:"mac"`
	Room string `json:"room"`
}

type Reading struct {
	ID    string      `json:"id,omitempty"`
	Name  string      `json:"name"`
	Value interface{} `json:"value"` // It can contains any value
	Unit  string      `json:"unit"`
}

type Room struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
