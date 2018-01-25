package svcdb

// Conversation model
type Conversation struct {
	ID            int64  `json:"id"`
	MessageCount  int64  `json:"messageCount"`
	RecipientID   int64  `json:"recipient"`
	RecipientType string `json:"type"` // Either a Group or a User
}

// Conversations Conversation array
type Conversations []Conversation

func (i *Conversation) NodeToConversation(nodes []interface{}) {
	i.ID = nodes[0].(int64)
	i.RecipientID = nodes[1].(int64)
	i.MessageCount = nodes[2].(int64)
}
