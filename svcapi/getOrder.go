package svcapi

import (
	"context"
	"encoding/json"
	"net/http"
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
	order, err := s.svcpayment.GetOrder(ctx, orderID)
	return order, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type getOrderRequest struct {
	OrderID int64 `json:"id"`
}

type getOrderResponse struct {
	Order svcdb.Order  `json:"order"`
}

func GetOrderEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getOrderRequest)
		order, err := svc.GetOrder(ctx, req.OrderID)
		return getOrderResponse{order}, err
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
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetOrder", logger), jwt.HTTPToContext()))...,
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

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) GetOrder(ctx context.Context, orderID int64) (svcdb.Order, error) {
	return mw.next.GetOrder(ctx, orderID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetOrder(ctx context.Context, orderID int64) (svcdb.Order, error) {
	return mw.next.GetOrder(ctx, orderID)
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
