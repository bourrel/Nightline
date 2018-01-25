package svcws

import (
	"context"
	"errors"
	"svcevent"

	"net/http"
	"svcdb"

	"github.com/Shopify/sarama"
	"github.com/go-kit/kit/log"
)

/* Service interface */
type IService interface {
	OpenConnection(h http.Handler, logger log.Logger, message chan *sarama.ConsumerMessage) http.Handler
	LastMessage(ctx context.Context, v interface{}) (lastMessageResponse, error)
	NewMessage(ctx context.Context, v interface{}) (interface{}, error)
}

/* Errors definition */
var (
	ConnError     = errors.New("The server was acting as a gateway or proxy and received an invalid response from the upstream server.")
	RequestError  = errors.New("The server cannot or will not process the request due to an apparent client error.")
	AuthError     = errors.New("The user might not have the necessary permissions for a resource, or may need an account of some sort.")
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
func NewService(db svcdb.IService, event svcevent.IService) IService {
	return Service{
		svcdb:    db,
		svcevent: event,
	}
}

type Service struct {
	svcdb    svcdb.IService
	svcevent svcevent.IService
}

/* Middleware interface */
type Middleware func(IService) IService
