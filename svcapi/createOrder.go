package svcapi

import (
	"context"
	"encoding/json"
	"net/http"

	"time"

	"github.com/go-kit/kit/auth/jwt"
	"github.com/go-kit/kit/tracing/opentracing"
	mux "github.com/gorilla/mux"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"

	"svcdb"
)
 
/*************** Service ***************/
func (s Service) CreateOrder(ctx context.Context, o svcdb.Order) (svcdb.Order, error) {
	order, err := s.svcpayment.CreateOrder(ctx, o)
	return order, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type createOrderRequest struct {
	Order	svcdb.Order `json:"order"`
}

type createOrderResponse struct {
	Order	svcdb.Order  `json:"order"`
}

func CreateOrderEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createOrderRequest)
		order, err := svc.CreateOrder(ctx, req.Order)
		return createOrderResponse{Order: order}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPCreateOrderRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func DecodeHTTPCreateOrderResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response createOrderResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func CreateOrderHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/orders").Handler(httptransport.NewServer(
		endpoints.CreateOrderEndpoint,
		DecodeHTTPCreateOrderRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "CreateOrder", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) CreateOrder(ctx context.Context, o svcdb.Order) (svcdb.Order, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "createOrder",
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.CreateOrder(ctx, o)
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) CreateOrder(ctx context.Context, o svcdb.Order) (svcdb.Order,error) {
	return mw.next.CreateOrder(ctx, o)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) CreateOrder(ctx context.Context, o svcdb.Order) (svcdb.Order, error) {
	return mw.next.CreateOrder(ctx, o)
}

/*************** Main ***************/
/* Main */
func BuildCreateOrderEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "CreateOrder")
		csLogger := log.With(logger, "method", "CreateOrder")

		csEndpoint = CreateOrderEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "CreateOrder")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
