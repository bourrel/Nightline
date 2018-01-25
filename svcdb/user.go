package svcdb

import (
	"encoding/json"
	"time"

	"github.com/johnnadratowski/golang-neo4j-bolt-driver/structures/graph"
)

// User model
type User struct {
	ID            int64     `json:"id"`
	Email         string    `json:"email"`
	Pseudo        string    `json:"pseudo"`
	Password      string    `json:"password"`
	Birthdate     time.Time `json:"birthdate,omitempty"`
	Firstname     string    `json:"firstname,omitempty"`
	Surname       string    `json:"surname,omitempty"`
	Number        string    `json:"number,omitempty"`
	Image         string    `json:"image,omitempty"`
	SuccessPoints int64     `json:"success_points,omitempty"`
	ConnectedTo   []User    `json:"connected_to,omitempty"`
	StripeID      string    `json:"stripeID,omitempty"`
}

// Users User array
type Users []User

func (u *User) NodeToUser(node graph.Node) {
	u.ID = node.NodeIdentity
	u.Email = node.Properties["Email"].(string)
	u.Pseudo = node.Properties["Pseudo"].(string)
	u.Password = node.Properties["Password"].(string)

	if node.Properties["Fistname"] != nil {
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
	// if node.Properties["ConnectedTo"] != nil {
	// 	u.ConnectedTo = node.Properties["ConnectedTo"].(string)
	// }
	if node.Properties["Birthdate"] != nil {
		u.Birthdate, _ = time.Parse(timeForm, node.Properties["Birthdate"].(string))
	}
	if node.Properties["StripeID"] != nil {
		u.StripeID = node.Properties["StripeID"].(string)
	}
}

func (u *User) MarshalJSON() ([]byte, error) {
	type Alias User

	return json.Marshal(&struct {
		*Alias
		Birthdate string `json:"birthdate"`
	}{
		Alias:     (*Alias)(u),
		Birthdate: u.Birthdate.Format(timeForm),
	})
}

func (u *User) UnmarshalJSON(data []byte) error {
	type Alias User

	aux := &struct {
		Birthdate string `json:"birthdate"`
		*Alias
	}{
		Alias: (*Alias)(u),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	u.Birthdate, _ = time.Parse(timeForm, aux.Birthdate)
	return nil
}

func (u *User) UpdateUser(new User) {
	if new.Email != "" {
		u.Email = new.Email
	}
	if new.Pseudo != "" {
		u.Pseudo = new.Pseudo
	}
	if new.Password != "" {
		u.Password = new.Password
	}
	if new.Birthdate.IsZero() {
		u.Birthdate = new.Birthdate
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
	if new.SuccessPoints >= 0 {
		u.SuccessPoints = new.SuccessPoints
	}
	// if new.ConnectedTo != "" {
	// 	u.ConnectedTo = new.ConnectedTo
	// }
	if new.StripeID != "" {
		u.StripeID = new.StripeID
	}
}

func (u *User) Complete() bool {
	if u.ID != 0 && u.Email != "" && u.Pseudo != "" && u.Password != "" &&
		u.Firstname != "" && u.Surname != "" && u.Number != "" {
		return true
	}
	return false
}
