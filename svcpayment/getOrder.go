package svcpayment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"strconv"
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
func (s Service) GetOrder(ctx context.Context, orderID int64) (svcdb.Order, error) {
	return s.svcdb.GetOrder(ctx, orderID)
}

/*************** Endpoint ***************/
type getOrderRequest struct {
	OrderID int64 `json:"id"`
}

type getOrderResponse struct {
	Order svcdb.Order  `json:"order"`
	Err   string `json:"err,omitempty"`
}

func GetOrderEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getOrderRequest)
		order, err := svc.GetOrder(ctx, req.OrderID)
		if err != nil {
			return getOrderResponse{order, err.Error()}, nil
		}
		return getOrderResponse{order, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetOrderRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getOrderRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	orderID, err := strconv.ParseInt(mux.Vars(r)["OrderID"], 10, 64)
	if err != nil {
		return nil, err
	}
	(&request).OrderID = orderID

	return request, nil
}

func DecodeHTTPGetOrderResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getOrderResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetOrderHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/orders/{OrderID:[0-9]+}").Handler(httptransport.NewServer(
		endpoints.GetOrderEndpoint,
		DecodeHTTPGetOrderRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetOrder", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetOrder(ctx context.Context, orderID int64) (svcdb.Order, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getOrder",
			"orderID", orderID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetOrder(ctx, orderID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetOrder(ctx context.Context, orderID int64) (svcdb.Order, error) {
	v, err := mw.next.GetOrder(ctx, orderID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetOrderEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetOrder")
		csLogger := log.With(logger, "method", "GetOrder")

		csEndpoint = GetOrderEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetOrder")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetOrder(ctx context.Context, orderID int64) (svcdb.Order, error) {
	var order svcdb.Order

	request := getOrderRequest{OrderID: orderID}
	response, err := e.GetOrderEndpoint(ctx, request)
	if err != nil {
		return order, err
	}
	order = response.(getOrderResponse).Order
	return order, str2err(response.(getOrderResponse).Err)
}

func EncodeHTTPGetOrderRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(getOrderRequest).OrderID)
	encodedUrl, err := route.Path(r.URL.Path).URL("OrderID", id)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetOrder(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/orders/{OrderID}"),
		EncodeHTTPGetOrderRequest,
		DecodeHTTPGetOrderResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "GetOrder")(ceEndpoint)
	return ceEndpoint, nil
}
