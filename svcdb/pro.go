package svcdb

import (
	"github.com/johnnadratowski/golang-neo4j-bolt-driver/structures/graph"
)

// Pro model
type Pro struct {
	ID             int64   `json:"id"`
	Email          string  `json:"email"`
	Pseudo         string  `json:"pseudo"`
	Password       string  `json:"password"`
	Firstname      string  `json:"firstname,omitempty"`
	Surname        string  `json:"surname,omitempty"`
	Number         string  `json:"number,omitempty"`
	Image          string  `json:"image"`
	StripeID	   string  `json:"stripeid"`
	StripeSKey	   string  `json:"stripeskey"`
	StripePKey	   string  `json:"stripepkey"`
	Establishments []int64 `json:"establishments"`
}

// Pros Pro array
type Pros []Pro

func (u *Pro) NodeToPro(node graph.Node) {
	u.ID = node.NodeIdentity
	u.Email = node.Properties["Email"].(string)
	u.Pseudo = node.Properties["Pseudo"].(string)
	u.Password = node.Properties["Password"].(string)

	if node.Properties["Firstname"] != nil {
		u.Firstname = node.Properties["Firstname"].(string)
	}
	if node.Properties["Surname"] != nil {
		u.Surname = node.Properties["Surname"].(string)
	}
	if node.Properties["Number"] != nil {
		u.Number = node.Properties["Number"].(string)
	}
	if node.Properties["Image"] != nil {
		u.Image = node.Properties["Image"].(string)
	}
}

/* USE WITH CAUTION, ALWAYS ENCRYPTED OR INTERNALLY, sensitive data inside */
func (u *Pro) NodeToProStripe(node graph.Node) {
	u.ID = node.NodeIdentity
	u.Email = node.Properties["Email"].(string)
	u.Pseudo = node.Properties["Pseudo"].(string)
	u.Password = node.Properties["Password"].(string)

	u.StripeID = node.Properties["StripeID"].(string)
	u.StripeSKey = node.Properties["StripeSKey"].(string)
	u.StripePKey = node.Properties["StripePKey"].(string)

	if node.Properties["Firstname"] != nil {
		u.Firstname = node.Properties["Firstname"].(string)
	}
	if node.Properties["Surname"] != nil {
		u.Surname = node.Properties["Surname"].(string)
	}
	if node.Properties["Number"] != nil {
		u.Number = node.Properties["Number"].(string)
	}
	if node.Properties["Image"] != nil {
		u.Image = node.Properties["Image"].(string)
	}
}

func (u *Pro) UpdatePro(new Pro) {
	if new.Email != "" {
		u.Email = new.Email
	}
	if new.Pseudo != "" {
		u.Pseudo = new.Pseudo
	}
	if new.Password != "" {
		u.Password = new.Password
	}

	if new.Firstname != "" {
		u.Firstname = new.Firstname
	}
	if new.Surname != "" {
		u.Surname = new.Surname
	}
	if new.Number != "" {
		u.Number = new.Number
	}
	if new.Image != "" {
		u.Image = new.Image
	}
}
