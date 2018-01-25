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
func (s Service) GetOrdersBySoiree(ctx context.Context, soireeID int64) ([]svcdb.Order, error) {
	orders, err := s.svcdb.GetOrdersBySoiree(ctx, soireeID)
	return orders, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type getOrdersBySoireeRequest struct {
	SoireeID int64 `json:"id"`
}

type getOrdersBySoireeResponse struct {
	Orders []svcdb.Order `json:"orders"`
}

func GetOrdersBySoireeEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getOrdersBySoireeRequest)
		soirees, err := svc.GetOrdersBySoiree(ctx, req.SoireeID)
		if err != nil {
			return getOrdersBySoireeResponse{soirees}, err
		}
		return getOrdersBySoireeResponse{soirees}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetOrdersBySoireeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getOrdersBySoireeRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	soireeID, err := strconv.ParseInt(mux.Vars(r)["SoireeID"], 10, 64)
	if err != nil {
		return nil, RequestError
	}
	(&request).SoireeID = soireeID

	return request, nil
}

func DecodeHTTPGetOrdersBySoireeResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getOrdersBySoireeResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetOrdersBySoireeHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/soiree/{SoireeID}/orders").Handler(httptransport.NewServer(
		endpoints.GetOrdersBySoireeEndpoint,
		DecodeHTTPGetOrdersBySoireeRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetOrdersBySoiree", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetOrdersBySoiree(ctx context.Context, soireeID int64) ([]svcdb.Order, error) {
	orders, err := mw.next.GetOrdersBySoiree(ctx, soireeID)

	mw.logger.Log(
		"method", "getOrdersBySoiree",
		"soireeID", soireeID,
		"response", orders,
		"took", time.Since(time.Now()),
	)
	return orders, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) GetOrdersBySoiree(ctx context.Context, soireeID int64) ([]svcdb.Order, error) {
	return mw.next.GetOrdersBySoiree(ctx, soireeID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetOrdersBySoiree(ctx context.Context, soireeID int64) ([]svcdb.Order, error) {
	return mw.next.GetOrdersBySoiree(ctx, soireeID)
}

/*************** Main ***************/
/* Main */
func BuildGetOrdersBySoireeEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetOrdersBySoiree")
		csLogger := log.With(logger, "method", "GetOrdersBySoiree")

		csEndpoint = GetOrdersBySoireeEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetOrdersBySoiree")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetOrdersBySoiree(ctx context.Context, soireeID int64) ([]svcdb.Order, error) {
	var et []svcdb.Order

	request := getOrdersBySoireeRequest{SoireeID: soireeID}
	response, err := e.GetOrdersBySoireeEndpoint(ctx, request)
	if err != nil {
		return et, err
	}
	et = response.(getOrdersBySoireeResponse).Orders
	return et, err
}

func EncodeHTTPGetOrdersBySoireeRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(getOrdersBySoireeRequest).SoireeID)
	encodedUrl, err := route.Path(r.URL.Path).URL("SoireeID", id)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetOrdersBySoiree(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/ordersBySoiree/{SoireeID}"),
		EncodeHTTPGetOrdersBySoireeRequest,
		DecodeHTTPGetOrdersBySoireeResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "GetOrdersBySoiree")(ceEndpoint)
	return ceEndpoint, nil
}
