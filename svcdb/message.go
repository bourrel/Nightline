package svcdb

import (
	"time"

	"github.com/johnnadratowski/golang-neo4j-bolt-driver/structures/graph"
)

// Message model
type Message struct {
	ID   int64     `json:"id"`
	From int64     `json:"from"`
	To   int64     `json:"to"`
	Date time.Time `json:"date"`
	Text string    `json:"message"`
}

// Messages Message array
type Messages []Message

func (i *Message) NodeToMessage(node graph.Node) {
	i.ID = node.NodeIdentity

	if date, ok := node.Properties["Date"].(string); ok {
		i.Date, _ = time.Parse(timeForm, date)
	}
	i.Text = node.Properties["Text"].(string)
}
