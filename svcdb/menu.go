package svcdb

import (
	"github.com/johnnadratowski/golang-neo4j-bolt-driver/structures/graph"
)

// Menu model
type Menu struct {
	ID            int64   `json:"id"`
	Desc          string  `json:"desc"`
	Name          string  `json:"name"`
	Consommations []Conso `json:"consos"`
}

// Menus Menu array
type Menus []Menu

func (m *Menu) NodeToMenu(node graph.Node) {
	m.ID = node.NodeIdentity
	m.Name = node.Properties["Name"].(string)
	m.Desc = node.Properties["Desc"].(string)
}
