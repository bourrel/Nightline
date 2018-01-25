package svcpayment

import (
	"svcdb"
)

type orderProgressNotif struct {
	Order	svcdb.Order	`json:"order"`
	UserID  int64		`json:"userID"`
	Step	string		`json:"step"`
	Message	string		`json:"message"`
}
