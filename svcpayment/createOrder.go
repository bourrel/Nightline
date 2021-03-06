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
func (s Service) CreateOrder(ctx context.Context, o svcdb.Order) (svcdb.Order, error) {
	order, err := s.svcdb.CreateOrder(ctx, o)
	if err != nil {
		fmt.Println("CreateOrder (CreateOrder) : " + err.Error())
		return order, err
	}
	order, err = s.PutOrder(ctx, order.ID, "Issued", true)
	if err != nil {
		fmt.Println("CreateOrder (PutOrderIssued) : " + err.Error())
		return order, err
	}
	return order, err
}

/*************** Endpoint ***************/
type createOrderRequest struct {
	Order	svcdb.Order `json:"order"`
}

type createOrderResponse struct {
	Order	svcdb.Order  `json:"order"`
	Err		string `json:"err,omitempty"`
}

func CreateOrderEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createOrderRequest)
		order, err := svc.CreateOrder(ctx, req.Order)

		// Create node
		if err != nil {
			fmt.Println("Error CreateOrderEndpoint 1 : ", err.Error())
			return createOrderResponse{Order: order, Err: err.Error()}, nil
		}

		return createOrderResponse{Order: order, Err: ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPCreateOrderRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		fmt.Println("Error DecodeHTTPCreateOrderRequest : ", err.Error(), err)
		return nil, err
	}
	return request, nil
}

func DecodeHTTPCreateOrderResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response createOrderResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPCreateOrderResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func CreateOrderHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/orders").Handler(httptransport.NewServer(
		endpoints.CreateOrderEndpoint,
		DecodeHTTPCreateOrderRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "CreateOrder", logger)))...,
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

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) CreateOrder(ctx context.Context, o svcdb.Order) (svcdb.Order, error) {
	v, err := mw.next.CreateOrder(ctx, o)
	mw.ints.Add(1)
	return v, err
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
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
func (e Endpoints) CreateOrder(ctx context.Context, o svcdb.Order) (svcdb.Order, error) {
	request := createOrderRequest{Order: o}
	response, err := e.CreateOrderEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error CreateOrder : ", err.Error())
		return o, err
	}
	return response.(createOrderResponse).Order, str2err(response.(createOrderResponse).Err)
}

func ClientCreateOrder(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, "/orders"),
		EncodeHTTPGenericRequest,
		DecodeHTTPCreateOrderResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "CreateOrder")(ceEndpoint)
	return ceEndpoint, nil
}
