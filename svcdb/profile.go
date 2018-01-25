package svcdb

import (
	"encoding/json"
	"time"

	"github.com/johnnadratowski/golang-neo4j-bolt-driver/structures/graph"
)

// Profile model
type Profile struct {
	ID            int64     `json:"id"`
	Email         string    `json:"email"`
	Pseudo        string    `json:"pseudo"`
	Firstname     string    `json:"firstname,omitempty"`
	Birthdate     time.Time `json:"birthdate,omitempty"`
	Surname       string    `json:"surname,omitempty"`
	Number        string    `json:"number,omitempty"`
	Image         string    `json:"image,omitempty"`
	FriendCount   int64     `json:"friend_count,omitempty"`
	SuccessPoints int64     `json:"success_points,omitempty"`
	ConnectedTo   []Profile `json:"connected_to,omitempty"`
	StripeID      string    `json:"stripeID,omitempty"`
}

// Profiles Profile array
type Profiles []Profile

func (u *Profile) NodeToProfile(node graph.Node) {
	u.ID = node.NodeIdentity
	u.Email = node.Properties["Email"].(string)
	u.Pseudo = node.Properties["Pseudo"].(string)

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
	if node.Properties["SuccessPoints"] != nil {
		u.SuccessPoints = node.Properties["SuccessPoints"].(int64)
	}
	if node.Properties["FriendCount"] != nil {
		u.SuccessPoints = node.Properties["FriendCount"].(int64)
	}
	// if node.Properties["ConnectedTo"] != "" {
	// 	u.ConnectedTo = node.Properties["ConnectedTo"].(string)
	// }
	if node.Properties["Birthdate"] != nil {
		u.Birthdate, _ = time.Parse(timeForm, node.Properties["Birthdate"].(string))
	}
	if node.Properties["StripeID"] != nil {
		u.StripeID = node.Properties["StripeID"].(string)
	}
}

func (p *Profile) MarshalJSON() ([]byte, error) {
	type Alias Profile

	return json.Marshal(&struct {
		*Alias
		Birthdate string `json:"birthdate"`
	}{
		Alias:     (*Alias)(p),
		Birthdate: p.Birthdate.Format(timeForm),
	})
}

func (p *Profile) UnmarshalJSON(data []byte) error {
	type Alias Profile

	aux := &struct {
		Birthdate string `json:"birthdate"`
		*Alias
	}{
		Alias: (*Alias)(p),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	p.Birthdate, _ = time.Parse(timeForm, aux.Birthdate)
	return nil
}
