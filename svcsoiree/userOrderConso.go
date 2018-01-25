package svcsoiree

import (
	"context"
	"encoding/json"
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
)

/*************** Service ***************/
/* Service - Business logic */
func (s Service) UserOrderConso(_ context.Context, userID, soireeID, consoID int64) (int64, error) {
	var orderID int64

	// check if user in soiree
	if false {
		return 0, Err
	}
	// check if conso in soiree->menu

	// proceed
	orderID = 10

	return orderID, nil
}

// Merged : Service definition

/*************** Endpoint ***************/
/* Endpoint - Req/Resp */
type userOrderConsoRequest struct{ UserID, SoireeID, ConsoID int64 }
type userOrderConsoResponse struct {
	OrderID	 int64
	Err      error
}

/* Endpoint - Create endpoint */
func UserOrderConsoEndpoint(s IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		uocReq := request.(userOrderConsoRequest)
		orderID, err := s.UserOrderConso(ctx, uocReq.UserID, uocReq.SoireeID, uocReq.ConsoID)
		return userOrderConsoResponse{
			OrderID:  orderID, 
			Err:      err,
		}, nil
	}
}

// Merged : endpoints struct

/*************** Transport ***************/
/* Transport - *coder Request */
func DecodeHTTPUserOrderConsoRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req userOrderConsoRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

/* Transport - *coder Response */
func DecodeHTTPUserOrderConsoResponse(_ context.Context, r *http.Response) (interface{}, error) {
	if r.StatusCode != http.StatusOK {
		return nil, errorDecoder(r)
	}
	var resp userOrderConsoResponse
	err := json.NewDecoder(r.Body).Decode(&resp)
	return resp, err
}

func UserOrderConsoHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/userOrderConso").Handler(httptransport.NewServer(
		endpoints.UserOrderConsoEndpoint,
		DecodeHTTPUserOrderConsoRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "UserOrderConso", logger)))...,
	))
	return route
}

// Merged : HTTPHandler, errorWrapper struct */

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) UserOrderConso(ctx context.Context, userID, soireeID, consoID int64) (orderID int64, err error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "userOrderConso",
			"userID", userID, "soireeID", soireeID, "consoID", consoID,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.UserOrderConso(ctx, userID, soireeID, consoID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) UserOrderConso(ctx context.Context, userID, soireeID, consoID int64) (int64, error) {
	v, err := mw.next.UserOrderConso(ctx, userID, soireeID, consoID)
	mw.userOrderConso_all.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildUserOrderConsoEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var uocEndpoint endpoint.Endpoint
	{
		uocDuration := duration.With("method", "UserOrderConso")
		uocLogger := log.With(logger, "method", "UserOrderConso")

		uocEndpoint = UserOrderConsoEndpoint(svc)
		uocEndpoint = opentracing.TraceServer(tracer, "UserOrderConso")(uocEndpoint)
		uocEndpoint = EndpointLoggingMiddleware(uocLogger)(uocEndpoint)
		uocEndpoint = EndpointInstrumentingMiddleware(uocDuration)(uocEndpoint)
	}
	return uocEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) UserOrderConso(ctx context.Context, userID, soireeID, consoID int64) (int64, error) {
	var orderID int64

	request := userOrderConsoRequest{
		UserID: userID,
		SoireeID: soireeID,
		ConsoID: consoID,
	}
	response, err := e.UserOrderConsoEndpoint(ctx, request)
	if err != nil {
		return 0, err
	}
	orderID = response.(userOrderConsoResponse).OrderID
	return orderID, response.(userOrderConsoResponse).Err
}

func ClientUserOrderConso(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, "/userOrderConso"),
		EncodeHTTPGenericRequest,
		DecodeHTTPUserOrderConsoResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "UserOrderConso")(gefmEndpoint)
	return gefmEndpoint, nil
}
