package svcsoiree

import (
	"context"
	"errors"

	"svcdb"
)

/* Service interface */
type IService interface {
	CreateSoiree(ctx context.Context, establishmentID, menuID int64, soiree svcdb.Soiree) (int64, error)
	UserOrderConso(ctx context.Context, userID int64, soireeID int64, consoID int64) (int64, error)
}

/* Errors definition */
var (
	Err = errors.New("One basic error")
	SoireeDateErr = errors.New("Error on soiree dates")
	InvalidSoireeErr = errors.New("Invalid soiree parameters provided")
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
func NewService(db svcdb.IService) IService {
	return Service{svcdb: db}
}

type Service struct {
	svcdb svcdb.IService
}

/* Middleware interface */
type Middleware func(IService) IService
