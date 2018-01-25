package client

import (
	"net/url"
	"strings"

	"github.com/go-kit/kit/log"
	stdopentracing "github.com/opentracing/opentracing-go"

	"svcsoiree"
)

func New(instance string, tracer stdopentracing.Tracer, logger log.Logger) (svcsoiree.IService, error) {
	if !strings.HasPrefix(instance, "http") {
		instance = "http://" + instance
	}
	u, err := url.Parse(instance)
	if err != nil {
		return nil, err
	}

	clientCreateSoiree, err := svcsoiree.ClientCreateSoiree(u, logger, tracer)
	clientUserOrderConso, err := svcsoiree.ClientUserOrderConso(u, logger, tracer)

	return svcsoiree.Endpoints{
		CreateSoireeEndpoint:  clientCreateSoiree,
		UserOrderConsoEndpoint:  clientUserOrderConso,
	}, nil

}
