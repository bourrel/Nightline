package svcdb

import (
	"strings"
	"encoding/json"
	time "time"

	"github.com/johnnadratowski/golang-neo4j-bolt-driver/structures/graph"
)

// UserOrder model
type UserOrder struct {
	User      User   `json:"user"`
	Price     int64 `json:"price"`
	Reference string `json:"reference"`
	Approved  string `json:"approved"`
}

// UserOrder array
type UserOrders []UserOrder

// ConsoOrder model
type ConsoOrder struct {
	Conso  Conso `json:"conso"`
	Amount int64 `json:"amount"`
}

// ConsoOrder array
type ConsoOrders []ConsoOrder

// StepOrder model
type StepOrder struct {
	ID     int64     `json:"id"`
	Name   string    `json:"name"`
	Date   time.Time `json:"date"`
	Result string    `json:"result"`
}

// StepOrder map
type StepOrders []StepOrder

var nextAllowedSteps = map[string]string{
	"Issued":      "Confirmed",
	"Confirmed":   "Verified",
	"Verified":    "Ready",
	"Ready":       "Deliverpaid",
	"Deliverpaid": "Completed",
	"Completed":   "",
}

func getNextAllowedStep(order Order) (string, error) {
	next := "Issued"
	stepMap := make(map[string]StepOrder)

	for i := range order.Steps {
		stepMap[order.Steps[i].Name] = order.Steps[i]
	}
	if len(order.Done) > 0 {
		return "", nil // No cont if order is finished
	}

	for len(next) > 0 {
		nextnext := nextAllowedSteps[next]
		if step, ok := stepMap[next]; ok {
			if len(step.Result) == 0 {
				return "", nil // No cont if step hasn't ended
			} else if step.Result == "false" {
				return "", nil // No cont if order has been stopped
			}
			_, ok := stepMap[nextnext]
			if len(nextnext) > 0 && !ok && len(step.Result) > 0 {
				return nextnext, nil // Success, found step finished and next one not started
			}
			// if this step done and nextstep present, continue
			next = nextnext
		}
	}
	return next, nil
}

func (so *StepOrder) MarshalJSON() ([]byte, error) {
	type Alias StepOrder
	return json.Marshal(&struct {
		*Alias
		Date string `json:"date"`
	}{
		Alias: (*Alias)(so),
		Date:  so.Date.Format(timeForm),
	})
}

func (so *StepOrder) UnmarshalJSON(data []byte) error {
	type Alias StepOrder
	aux := &struct {
		Date string `json:"date"`
		*Alias
	}{
		Alias: (*Alias)(so),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if len(aux.Date) > 0 {
		date, err := time.Parse(timeForm, aux.Date)
		if err != nil {
			date, err = time.Parse(timeFormFucked, aux.Date)
			if err != nil {
				return err
			}
		}
		so.Date = date
	}
	return nil
}

// Order model
type Order struct {
	ID     int64       `json:"id"`
	Done   string      `json:"done"`
	Price  int64      `json:"price"`
	Soiree Soiree      `json:"soiree"`
	Users  UserOrders  `json:"users"`
	Consos ConsoOrders `json:"consos"`
	Steps  StepOrders  `json:"steps"`
}

// Orders Order array
type Orders []Order

func (o *Order) NodeToOrder(node graph.Node) {
	o.ID = node.NodeIdentity
	o.Price = node.Properties["Price"].(int64)

	if node.Properties["Done"] != nil {
		o.Done = node.Properties["Done"].(string)
	}
}

func (o *Order) MarshalJSON() ([]byte, error) {
	type Alias Order
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(o),
	})
}

func (o *Order) UnmarshalJSON(data []byte) error {
	type Alias Order
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(o),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	return nil
}

func (o *Order) RelationToSoiree(orderNode graph.Node, relation graph.Relationship, soireeNode graph.Node) {
	var soiree Soiree

	soiree.NodeToSoiree(soireeNode)
	o.Soiree = soiree
}

func (o *Order) RelationAddUser(orderNode graph.Node, relation graph.Relationship, userNode graph.Node) {
	var user User
	var reference, approved string
	var price int64

	user.NodeToUser(userNode)
	price = relation.Properties["Price"].(int64)
	reference, _ = relation.Properties["Reference"].(string)
	approved = relation.Properties["Approved"].(string)

	approved = strings.ToLower(approved)
	o.Users = append(o.Users, UserOrder{
		User:      user,
		Price:     price,
		Reference: reference,
		Approved:  approved,
	})
}

func (o *Order) RelationAddConso(orderNode graph.Node, relation graph.Relationship, consoNode graph.Node) {
	var conso Conso
	var amount int64

	conso.NodeToConso(consoNode)
	amount = relation.Properties["Amount"].(int64)
	o.Consos = append(o.Consos, ConsoOrder{
		Conso:  conso,
		Amount: amount,
	})
}

func (o *Order) RelationAddStep(orderNode graph.Node, relation graph.Relationship, stepNode graph.Node) {
	var id int64
	var name, result string
	var date time.Time

	id = stepNode.NodeIdentity
	name = stepNode.Properties["Name"].(string)
	date, _ = time.Parse(timeForm, stepNode.Properties["Date"].(string))
	result, _ = stepNode.Properties["Result"].(string)

	o.Steps = append(o.Steps, StepOrder{
		ID:     id,
		Name:   name,
		Date:   date,
		Result: result,
	})
}
