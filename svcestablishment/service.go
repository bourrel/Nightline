package svcestablishment

import (
	"context"
	"errors"
	"time"

	"svcdb"
	"svcpayment"
	"svcsoiree"
)

/* Service interface */
type IService interface {
	CreateSoiree(ctx context.Context, establishmentID, menuID int64, s svcdb.Soiree) (int64, error)
	DeliverOrder(ctx context.Context, establishmentID, menuID int64, soireeBegin, soireeEnd time.Time) (int64, error)
	GetOrder(ctx context.Context, orderID int64) (svcdb.Order, error)
	GetOrdersBySoiree(ctx context.Context, soireeID int64) ([]svcdb.Order, error)
	SearchOrders(ctx context.Context, order svcdb.Order) ([]svcdb.Order, error)
	PutOrder(ctx context.Context, orderID int64, step string, flag bool) (svcdb.Order, error)
	GetSoirees(ctx context.Context, estabID int64) ([]svcdb.Soiree, error)
	GetStat(ctx context.Context, establishmentID, menuID int64, soireeBegin, soireeEnd time.Time) (int64, error)
	LoginPro(ctx context.Context, old svcdb.Pro) (svcdb.Pro, string, error)
	RegisterPro(ctx context.Context, old svcdb.Pro) (svcdb.Pro, string, error)
	CreateEstab(ctx context.Context, estab svcdb.Establishment, proID int64) (svcdb.Establishment, error)
	UpdateEstab(ctx context.Context, establishment svcdb.Establishment) (svcdb.Establishment, error)
	DeleteEstab(ctx context.Context, estabID int64) error
	DeleteSoiree(ctx context.Context, soireeID int64) error
	UpdatePro(ctx context.Context, pro svcdb.Pro) (svcdb.Pro, error)
	CreateMenu(ctx context.Context, establishmentID int64, menu svcdb.Menu) (svcdb.Menu, error)
	CreateConso(ctx context.Context, establishmentID, menuID int64, conso svcdb.Conso) (svcdb.Conso, error)
	GetConso(ctx context.Context, estabID int64) ([]svcdb.Conso, error)
	GetConsoByOrderID(ctx context.Context, consoID int64) (svcdb.Conso, error)
	GetMenu(ctx context.Context, estabID int64) ([]svcdb.Menu, error)
	GetEstablishmentType(ctx context.Context) ([]string, error)
	GetSoireeOrders(ctx context.Context, estabID int64) ([]svcdb.Order, error)
	GetProEstablishments(ctx context.Context, estabID int64) ([]svcdb.Establishment, error)
	GetAnalyseP(ctx context.Context, estabID int64, soireeID int64) ([]svcdb.AnalyseP, error)
}

/* Errors definition */
var (
	ConnError     = errors.New("The server was acting as a gateway or proxy and received an invalid response from the upstream server")
	RequestError  = errors.New("The server cannot or will not process the request due to an apparent client error")
	AuthError     = errors.New("The user might not have the necessary permissions for a resource, or may need an account of some sort")
	NotFoundError = errors.New("The requested resource could not be found.")
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

func dbToHTTPErr(err error) error {
	if err == nil {
		return nil
	}
	switch err.Error() {
	case "EOF":
		return NotFoundError
	}
	return err
}

/* Service implementation */
func NewService(db svcdb.IService, soiree svcsoiree.IService, payment svcpayment.IService) IService {
	return Service{svcdb: db, svcsoiree: soiree, svcpayment: payment}
}

type Service struct {
	svcdb      svcdb.IService
	svcsoiree  svcsoiree.IService
	svcpayment svcpayment.IService
}

/* Middleware interface */
type Middleware func(IService) IService
