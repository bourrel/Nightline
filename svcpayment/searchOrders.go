package svcpayment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

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
func (s Service) SearchOrders(ctx context.Context, order svcdb.Order) (svcdb.Orders, error) {
	return s.svcdb.SearchOrders(ctx, order)
}

/*************** Endpoint ***************/
type searchOrdersRequest struct {
	Order		svcdb.Order	`json:"order"`
}

type searchOrdersResponse struct {
	Orders		svcdb.Orders	`json:"orders"`
	Err			string	`json:"err,omitempty"`
}

func SearchOrdersEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(searchOrdersRequest)
		orders, err := svc.SearchOrders(ctx, req.Order)
		if err != nil {
			fmt.Println("Error SearchOrdersEndpoint : ", err.Error())
			return searchOrdersResponse{orders, err.Error()}, nil
		}
		return searchOrdersResponse{orders, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPSearchOrdersRequest(_ context.Context, r *http.Request) (interface{}, error) {
    var request searchOrdersRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		fmt.Println("Error DecodeHTTPSearchOrdersRequest : ", err.Error(), err)
		return nil, err
	}
	return request, nil
}

func DecodeHTTPSearchOrdersResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response searchOrdersResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPSearchOrdersResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func SearchOrdersHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/search/orders").Handler(httptransport.NewServer(
		endpoints.SearchOrdersEndpoint,
		DecodeHTTPSearchOrdersRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "SearchOrders", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) SearchOrders(ctx context.Context, order svcdb.Order) (svcdb.Orders, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "searchOrders",
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.SearchOrders(ctx, order)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) SearchOrders(ctx context.Context, order svcdb.Order) (svcdb.Orders, error) {
	v, err := mw.next.SearchOrders(ctx, order)
	mw.ints.Add(1)
	return v, err
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
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forsearch limiter & circuitbreaker for now kthx
func (e Endpoints) SearchOrders(ctx context.Context, order svcdb.Order) (svcdb.Orders, error) {
	var orders svcdb.Orders

	request := searchOrdersRequest{Order: order}
	response, err := e.SearchOrdersEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error Client SearchOrders : ", err.Error())
		return orders, err
	}
	orders = response.(searchOrdersResponse).Orders
	return orders, str2err(response.(searchOrdersResponse).Err)
}

func ClientSearchOrders(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, "/search/orders"),
		EncodeHTTPGenericRequest,
		DecodeHTTPSearchOrdersResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "SearchOrders")(ceEndpoint)
	return ceEndpoint, nil
}
