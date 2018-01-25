package svcdb

import (
	"github.com/johnnadratowski/golang-neo4j-bolt-driver/structures/graph"
)

// EstablishmentType model
type EstablishmentType struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// EstablishmentTypes EstablishmentType array
type EstablishmentTypes []EstablishmentType

func (e *EstablishmentType) NodeToEstablishmentType(node graph.Node) {
	e.ID = node.NodeIdentity
	e.Name = node.Properties["Name"].(string)
}

func isEstablishmentType(estabType string) bool {
	// for _, currType := range EstablishmentTypeType {
	// 	if currType == estabType {
	// 		return true
	// 	}
	// }
	// return false
	return true
}
