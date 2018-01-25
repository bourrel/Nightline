package client

import (
	"net/url"
	"strings"

	"github.com/go-kit/kit/log"
	stdopentracing "github.com/opentracing/opentracing-go"

	"svcpayment"
)

func New(instance string, tracer stdopentracing.Tracer, logger log.Logger) (svcpayment.IService, error) {
	if !strings.HasPrefix(instance, "http") {
		instance = "http://" + instance
	}
	u, err := url.Parse(instance)
	if err != nil {
		return nil, err
	}

	/* Order */
	clientGetOrder, err := svcpayment.ClientGetOrder(u, logger, tracer)
	clientCreateOrder, err := svcpayment.ClientCreateOrder(u, logger, tracer)
	clientPutOrder, err := svcpayment.ClientPutOrder(u, logger, tracer)
	clientSearchOrders, err := svcpayment.ClientSearchOrders(u, logger, tracer)
	clientAnswerOrder, err := svcpayment.ClientAnswerOrder(u, logger, tracer)

	/* Pro */
	clientRegisterPro, err := svcpayment.ClientRegisterPro(u, logger, tracer)
	
    return svcpayment.Endpoints{
		/* Order */
		GetOrderEndpoint:				clientGetOrder,		
		CreateOrderEndpoint:			clientCreateOrder,		
		PutOrderEndpoint:				clientPutOrder,		
		SearchOrdersEndpoint:			clientSearchOrders,
		AnswerOrderEndpoint:			clientAnswerOrder,

		/* Pro */
		RegisterProEndpoint:			clientRegisterPro,
	}, nil
}
