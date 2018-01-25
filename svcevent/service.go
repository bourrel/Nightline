package svcevent

import (
	"context"
	"errors"
)

/* Service interface */
type IService interface {
	Push(ctx context.Context, name string, body interface{}, userID int64) (int32, int64, error)
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
func NewService() IService {
	return Service{}
}

type Service struct{}

/* Middleware interface */
type Middleware func(IService) IService
