package svcdb

import (
	"time"

	"github.com/johnnadratowski/golang-neo4j-bolt-driver/structures/graph"
)

// Invitation model
type Invitation struct {
	ID   int64     `json:"id"`
	From User      `json:"from"`
	To   User      `json:"to"`
	Date time.Time `json:"date"`
}

// Invitations Invitation array
type Invitations []Invitation

func (i *Invitation) RelationToInvitation(from graph.Node, relation graph.Relationship, to graph.Node) {
	var fromUser User
	var toUser User

	fromUser.NodeToUser(from)
	toUser.NodeToUser(to)

	i.ID = relation.RelIdentity
	i.From = fromUser
	i.To = toUser
	i.Date, _ = time.Parse(timeForm, relation.Properties["Date"].(string))
}
