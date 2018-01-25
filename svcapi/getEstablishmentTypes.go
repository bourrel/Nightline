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
)

/*************** Service ***************/
func (s Service) GetEstablishmentTypes(ctx context.Context) ([]string, error) {
	types, err := s.svcdb.GetEstablishmentTypes(ctx)
	return types, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type getEstablishmentTypesRequest struct {
}

type getEstablishmentTypesResponse struct {
	Establishments []string `json:"types"`
}

func GetEstablishmentTypesEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		estabs, err := svc.GetEstablishmentTypes(ctx)
		return getEstablishmentTypesResponse{Establishments: estabs}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPgetEstablishmentTypesRequest(_ context.Context, r *http.Request) (interface{}, error) {
	return nil, nil
}

func DecodeHTTPgetEstablishmentTypesResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getEstablishmentTypesResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, RequestError
	}
	return response, nil
}

func GetEstablishmentTypesHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/establishmentTypes").Handler(httptransport.NewServer(
		endpoints.GetEstablishmentTypesEndpoint,
		DecodeHTTPgetEstablishmentTypesRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetEstablishmentTypes", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetEstablishmentTypes(ctx context.Context) ([]string, error) {
	estabs, err := mw.next.GetEstablishmentTypes(ctx)

	mw.logger.Log(
		"method", "GetEstablishmentTypes",
		"response", estabs,
		"took", time.Since(time.Now()),
	)
	return estabs, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) GetEstablishmentTypes(ctx context.Context) ([]string, error) {
	return mw.next.GetEstablishmentTypes(ctx)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetEstablishmentTypes(ctx context.Context) ([]string, error) {
	return mw.next.GetEstablishmentTypes(ctx)
}

/*************** Main ***************/
/* Main */
func BuildGetEstablishmentTypesEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetEstablishmentTypes")
		csLogger := log.With(logger, "method", "GetEstablishmentTypes")

		csEndpoint = GetEstablishmentTypesEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetEstablishmentTypes")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
