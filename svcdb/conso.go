package svcdb

import (
	"github.com/johnnadratowski/golang-neo4j-bolt-driver/structures/graph"
)

// Conso model
type Conso struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
	Picture     string  `json:"picture,omitempty"`
}

// Consos Conso array
type Consos []Conso

func (c *Conso) NodeToConso(node graph.Node) {
	c.ID = node.NodeIdentity
	c.Name = node.Properties["Name"].(string)
	if price, ok := node.Properties["Price"].(float64); ok {
		c.Price = price
	} else if price, ok := node.Properties["Price"].(int64); ok {
		c.Price = float64(price)
	}
	if node.Properties["Desc"] != nil {
		c.Description = node.Properties["Desc"].(string)
	}
	if node.Properties["Picture"] != nil {
		c.Picture = node.Properties["Picture"].(string)
	}
}
