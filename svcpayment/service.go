package svcpayment

import (
	"context"
	"errors"

	"svcdb"
	"svcevent"
)

/* Service interface */
type IService interface {
	/* Order */
	GetOrder(ctx context.Context, orderID int64) (svcdb.Order, error)
	CreateOrder(ctx context.Context, order svcdb.Order) (svcdb.Order, error)
	PutOrder(ctx context.Context, orderID int64, step string, flag bool) (svcdb.Order, error)
 	SearchOrders(ctx context.Context, order svcdb.Order) (svcdb.Orders, error)
	AnswerOrder(ctx context.Context, orderID int64, userID int64, answer bool) (svcdb.Order, error)

	/* Pro */
	RegisterPro(ctx context.Context, p svcdb.Pro) (svcdb.Pro, error)
}

/* Errors definition */
var (
	Err          = errors.New("One basic error")
	RequestError = errors.New("The server cannot or will not process the request due to an apparent client error")
	DateErr      = errors.New("Date error, logic invalid operation")
)

/* Misc */
func str2err(s string) error {
	if s == "" {
		return nil
	}
	return errors.New(s)
}

func err2str(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

/* Service implementation */
func NewService(db svcdb.IService, event svcevent.IService) IService {
	return Service{
		svcdb: db,
		svcevent: event,
	}
}

type Service struct{
	svcdb		svcdb.IService
	svcevent    svcevent.IService
}

/* Middleware interface */
type Middleware func(IService) IService
