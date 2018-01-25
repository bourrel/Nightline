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
func (s Service) GetConsoByOrderID(ctx context.Context, orderID int64) (svcdb.Conso, error) {
	consos, err := s.svcdb.GetConsoByOrderID(ctx, orderID)
	return consos, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type GetConsoByOrderIDRequest struct {
	OrderID int64 `json:"id"`
}

type GetConsoByOrderIDResponse struct {
	Conso svcdb.Conso `json:"conso"`
	Err   error       `json:"err,omitempty"`
}

func GetConsoByOrderIDEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetConsoByOrderIDRequest)
		conso, err := svc.GetConsoByOrderID(ctx, req.OrderID)
		return GetConsoByOrderIDResponse{Conso: conso, Err: err}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetConsoByOrderIDRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request GetConsoByOrderIDRequest

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

func DecodeHTTPGetConsoByOrderIDResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response GetConsoByOrderIDResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, RequestError
	}
	return response, nil
}

func GetConsoByOrderIDHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/orders/{OrderID}/conso").Handler(httptransport.NewServer(
		endpoints.GetConsoByOrderIDEndpoint,
		DecodeHTTPGetConsoByOrderIDRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetConsoByOrderID", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetConsoByOrderID(ctx context.Context, orderID int64) (svcdb.Conso, error) {
	conso, err := mw.next.GetConsoByOrderID(ctx, orderID)

	mw.logger.Log(
		"method", "GetConsoByOrderID",
		"orderID", orderID,
		"response", conso,
		"took", time.Since(time.Now()),
	)
	return conso, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) GetConsoByOrderID(ctx context.Context, orderID int64) (svcdb.Conso, error) {
	return mw.next.GetConsoByOrderID(ctx, orderID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetConsoByOrderID(ctx context.Context, orderID int64) (svcdb.Conso, error) {
	v, err := mw.next.GetConsoByOrderID(ctx, orderID)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetConsoByOrderIDEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetConsoByOrderID")
		csLogger := log.With(logger, "method", "GetConsoByOrderID")

		csEndpoint = GetConsoByOrderIDEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetConsoByOrderID")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetConsoByOrderID(ctx context.Context, etID int64) (svcdb.Conso, error) {
	var conso svcdb.Conso

	request := GetConsoByOrderIDRequest{OrderID: etID}
	response, err := e.GetConsoByOrderIDEndpoint(ctx, request)
	if err != nil {
		return conso, err
	}
	conso = response.(GetConsoByOrderIDResponse).Conso
	return conso, response.(GetConsoByOrderIDResponse).Err
}

func EncodeHTTPGetConsoByOrderIDRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(GetConsoByOrderIDRequest).OrderID)
	encodedUrl, err := route.Path(r.URL.Path).URL("OrderID", id)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetConsoByOrderID(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/consos/{OrderID}"),
		EncodeHTTPGetConsoByOrderIDRequest,
		DecodeHTTPGetConsoByOrderIDResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "GetConsoByOrderID")(ceEndpoint)
	return ceEndpoint, nil
}
