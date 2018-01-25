package svcestablishment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
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
func (s Service) GetOrder(ctx context.Context, orderID int64) (svcdb.Order, error) {
	order, err := s.svcdb.GetOrder(ctx, orderID)
	return order, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type getOrderRequest struct {
	OrderID int64 `json:"id"`
}

type getOrderResponse struct {
	Order svcdb.Order `json:"order"`
	Err   error       `json:"err,omitempty"`
}

func GetOrderEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getOrderRequest)
		order, err := svc.GetOrder(ctx, req.OrderID)
		return getOrderResponse{Order: order, Err: err}, nil
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
		return nil, RequestError
	}
	(&request).OrderID = orderID

	return request, nil
}

func DecodeHTTPGetOrderResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getOrderResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, RequestError
	}
	return response, nil
}

func GetOrderHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/orders/{OrderID}").Handler(httptransport.NewServer(
		endpoints.GetOrderEndpoint,
		DecodeHTTPGetOrderRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetOrder", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetOrder(ctx context.Context, orderID int64) (svcdb.Order, error) {
	order, err := mw.next.GetOrder(ctx, orderID)

	mw.logger.Log(
		"method", "getOrder",
		"orderID", orderID,
		"response", order,
		"took", time.Since(time.Now()),
	)
	return order, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) GetOrder(ctx context.Context, orderID int64) (svcdb.Order, error) {
	return mw.next.GetOrder(ctx, orderID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetOrder(ctx context.Context, orderID int64) (svcdb.Order, error) {
	v, err := mw.next.GetOrder(ctx, orderID)
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
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetOrder(ctx context.Context, etID int64) (svcdb.Order, error) {
	var order svcdb.Order

	request := getOrderRequest{OrderID: etID}
	response, err := e.GetOrderEndpoint(ctx, request)
	if err != nil {
		return order, err
	}
	order = response.(getOrderResponse).Order
	return order, response.(getOrderResponse).Err
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
