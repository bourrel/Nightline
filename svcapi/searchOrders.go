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
func (s Service) SearchOrders(ctx context.Context, order svcdb.Order) ([]svcdb.Order, error) {
	orders, err := s.svcpayment.SearchOrders(ctx, order)
	return orders, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type searchOrdersRequest struct {
	Order		svcdb.Order	`json:"order"`
}

type searchOrdersResponse struct {
	Orders		[]svcdb.Order	`json:"orders"`
}

func SearchOrdersEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(searchOrdersRequest)
		orders, err := svc.SearchOrders(ctx, req.Order)
		return searchOrdersResponse{orders}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPSearchOrdersRequest(_ context.Context, r *http.Request) (interface{}, error) {
    var request searchOrdersRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func DecodeHTTPSearchOrdersResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response searchOrdersResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func SearchOrdersHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/search/orders").Handler(httptransport.NewServer(
		endpoints.SearchOrdersEndpoint,
		DecodeHTTPSearchOrdersRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "SearchOrders", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) SearchOrders(ctx context.Context, order svcdb.Order) ([]svcdb.Order, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "searchOrders",
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.SearchOrders(ctx, order)
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) SearchOrders(ctx context.Context, order svcdb.Order) ([]svcdb.Order, error) {
	return mw.next.SearchOrders(ctx, order)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) SearchOrders(ctx context.Context, order svcdb.Order) ([]svcdb.Order, error) {
	return mw.next.SearchOrders(ctx, order)
}

/*************** Main ***************/
/* Main */
func BuildSearchOrdersEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "SearchOrders")
		csLogger := log.With(logger, "method", "SearchOrders")

		csEndpoint = SearchOrdersEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "SearchOrders")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
