package svcdb

import (
	"github.com/johnnadratowski/golang-neo4j-bolt-driver/structures/graph"
)

// Establishment model
type Establishment struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Address     string  `json:"address"`
	Long        float64 `json:"long"`
	Lat         float64 `json:"lat"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Rate        float64 `json:"rate"`
	Image       string  `json:"image,omitempty"`
	OpenHours   string  `json:"open_hours,omitempty"`
	Owner       int64   `json:"owner"`
}

// Establishments Establishment array
type Establishments []Establishment

func (e *Establishment) NodeToEstablishment(node graph.Node) {
	e.ID = node.NodeIdentity
	e.Name = node.Properties["Name"].(string)
	e.Long = node.Properties["Long"].(float64)
	e.Lat = node.Properties["Lat"].(float64)

	if node.Properties["Address"] != nil {
		e.Address = node.Properties["Address"].(string)
	}
	if node.Properties["Type"] != nil {
		e.Type = node.Properties["Type"].(string)
	}
	if node.Properties["Description"] != nil {
		e.Description = node.Properties["Description"].(string)
	}
	if node.Properties["Image"] != nil {
		e.Image = node.Properties["Image"].(string)
	}
	if node.Properties["OpenHours"] != nil {
		e.OpenHours = node.Properties["OpenHours"].(string)
	}
}

func (u *Establishment) UpdateEstablishment(new Establishment) {
	if new.Name != "" {
		u.Name = new.Name
	}
	if new.Long != u.Long {
		u.Long = new.Long
	}
	if new.Lat != u.Lat {
		u.Lat = new.Lat
	}
	if new.Address != "" {
		u.Address = new.Address
	}
	if new.Type != u.Type {
		u.Type = new.Type
	}
	if new.Description != "" {
		u.Description = new.Description
	}
	if new.Image != "" {
		u.Image = new.Image
	}
	if new.OpenHours != "" {
		u.OpenHours = new.OpenHours
	}
	// if new.ConnectedTo != "" {
	// 	u.ConnectedTo = new.ConnectedTo
	// }
}
