package svcdb

import (
	"github.com/johnnadratowski/golang-neo4j-bolt-driver/structures/graph"
)

// Success model
type Success struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Value  string `json:"value"`
	Active bool   `json:"active"`
}

func (s *Success) NodeToSuccess(node graph.Node) {
	s.ID = node.NodeIdentity

	if val, ok := node.Properties["Name"]; ok {
		s.Name = val.(string)
	}

	if val, ok := node.Properties["Value"]; ok {
		s.Value = val.(string)
	}
}
