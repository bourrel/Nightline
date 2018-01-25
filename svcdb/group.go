package svcdb

import (
	"github.com/johnnadratowski/golang-neo4j-bolt-driver/structures/graph"
)

// Group model
type Group struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Users       []Profile `json:"users"`
	Owner       Profile   `json:"owner"`
}

type GroupArrayElement struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	UserCount   int64  `json:"userCount"`
}

// Groups Group array
type Groups []Group

func (g *Group) NodeToGroup(node graph.Node) {
	g.ID = node.NodeIdentity
	g.Name = node.Properties["Name"].(string)

	if node.Properties["Description"] != nil {
		g.Description = node.Properties["Description"].(string)
	}
}

func (g *GroupArrayElement) NodeToGroupArrayElement(node graph.Node) {
	g.ID = node.NodeIdentity
	g.Name = node.Properties["Name"].(string)

	if node.Properties["Description"] != nil {
		g.Description = node.Properties["Description"].(string)
	}
}

func (g *Group) UpdateFrom(new Group) {
	if new.Name != "" {
		g.Name = new.Name
	}
	if new.Description != g.Description {
		g.Description = new.Description
	}
}
