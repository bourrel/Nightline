package svcestablishment

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/go-kit/kit/auth/jwt"
	"github.com/go-kit/kit/tracing/opentracing"
	mux "github.com/gorilla/mux"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"
)

/*************** Service ***************/
func (s Service) GetEstablishmentType(ctx context.Context) ([]string, error) {
	types, err := s.svcdb.GetEstablishmentTypes(ctx)
	return types, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type getEstablishmentTypeRequest struct {
}

type getEstablishmentTypeResponse struct {
	EstablishmentTypes []string `json:"types"`
}

func GetEstablishmentTypeEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		EstablishmentTypes, err := svc.GetEstablishmentType(ctx)
		return getEstablishmentTypeResponse{EstablishmentTypes: EstablishmentTypes}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPGetEstablishmentTypeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	return nil, nil
}

func DecodeHTTPGetEstablishmentTypeResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getEstablishmentTypeResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetEstablishmentTypeHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/establishmentTypes").Handler(httptransport.NewServer(
		endpoints.GetEstablishmentTypeEndpoint,
		DecodeHTTPGetEstablishmentTypeRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetEstablishmentType", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetEstablishmentType(ctx context.Context) ([]string, error) {
	establishmentTypes, err := mw.next.GetEstablishmentType(ctx)

	mw.logger.Log(
		"method", "getEstablishmentType",
		"response", establishmentTypes,
		"took", time.Since(time.Now()),
	)
	return establishmentTypes, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) GetEstablishmentType(ctx context.Context) ([]string, error) {
	return mw.next.GetEstablishmentType(ctx)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetEstablishmentType(ctx context.Context) ([]string, error) {
	return mw.next.GetEstablishmentType(ctx)
}

/*************** Main ***************/
/* Main */
func BuildGetEstablishmentTypeEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetEstablishmentType")
		csLogger := log.With(logger, "method", "GetEstablishmentType")

		csEndpoint = GetEstablishmentTypeEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetEstablishmentType")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetEstablishmentType(ctx context.Context) ([]string, error) {
	var EstablishmentType []string

	request := getEstablishmentTypeRequest{}
	response, err := e.GetEstablishmentTypeEndpoint(ctx, request)
	if err != nil {
		return EstablishmentType, err
	}
	EstablishmentType = response.(getEstablishmentTypeResponse).EstablishmentTypes
	return EstablishmentType, err
}

func ClientGetEstablishmentType(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/EstablishmentTypes"),
		EncodeHTTPGenericRequest,
		DecodeHTTPGetEstablishmentTypeResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "GetEstablishmentType")(ceEndpoint)
	return ceEndpoint, nil
}
