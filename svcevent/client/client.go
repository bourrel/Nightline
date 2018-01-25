package client

import (
	"net/url"
	"strings"
	"svcevent"

	"github.com/go-kit/kit/log"
	stdopentracing "github.com/opentracing/opentracing-go"
)

func New(instance string, tracer stdopentracing.Tracer, logger log.Logger) (svcevent.IService, error) {
	if !strings.HasPrefix(instance, "http") {
		instance = "http://" + instance
	}
	u, err := url.Parse(instance)
	if err != nil {
		return nil, err
	}

	clientPush, err := svcevent.ClientPush(u, logger, tracer)

	return svcevent.Endpoints{
		PushEndpoint: clientPush,
	}, nil

}
