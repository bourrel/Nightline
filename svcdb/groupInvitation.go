package svcdb

import (
	"time"

	"github.com/johnnadratowski/golang-neo4j-bolt-driver/structures/graph"
)

// GroupInvitation model
type GroupInvitation struct {
	ID   int64     `json:"id"`
	From Group     `json:"from"`
	To   User      `json:"to"`
	Date time.Time `json:"date"`
}

// GroupInvitations GroupInvitation array
type GroupInvitations []GroupInvitation

func (i *GroupInvitation) RelationToGroupInvitation(from graph.Node, relation graph.Relationship, to graph.Node) {
	var fromGroup Group
	var toUser User

	fromGroup.NodeToGroup(from)
	toUser.NodeToUser(to)

	i.ID = relation.RelIdentity
	i.From = fromGroup
	i.To = toUser
	i.Date, _ = time.Parse(timeForm, relation.Properties["Date"].(string))
}
