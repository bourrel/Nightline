package svcapi

import (
	"context"
	"encoding/json"
	"fmt"
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

	"svcdb"
)

/*************** Service ***************/
func (s Service) GetAllEstablishments(ctx context.Context) ([]svcdb.Establishment, error) {
	estabs, err := s.svcdb.GetEstablishments(ctx)
	return estabs, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type getAllEstablishmentsRequest struct {
}

type getAllEstablishmentsResponse struct {
	Establishments []svcdb.Establishment `json:"establishments"`
}

func GetAllEstablishmentsEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		estabs, err := svc.GetAllEstablishments(ctx)
		if err != nil {
			fmt.Println("Error GetAllEstablishmentsEndpoint : ", err.Error())
			return nil, err
		}
		return getAllEstablishmentsResponse{Establishments: estabs}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPgetAllEstablishmentsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	return nil, nil
}

func DecodeHTTPgetAllEstablishmentsResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getAllEstablishmentsResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPgetAllEstablishmentsResponse : ", err.Error())
		return nil, RequestError
	}
	return response, nil
}

func GetAllEstablishmentsHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/establishments").Handler(httptransport.NewServer(
		endpoints.GetAllEstablishmentsEndpoint,
		DecodeHTTPgetAllEstablishmentsRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetAllEstablishments", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetAllEstablishments(ctx context.Context) ([]svcdb.Establishment, error) {
	estabs, err := mw.next.GetAllEstablishments(ctx)

	mw.logger.Log(
		"method", "GetAllEstablishments",
		"response", estabs,
		"took", time.Since(time.Now()),
	)
	return estabs, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) GetAllEstablishments(ctx context.Context) ([]svcdb.Establishment, error) {
	return mw.next.GetAllEstablishments(ctx)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetAllEstablishments(ctx context.Context) ([]svcdb.Establishment, error) {
	return mw.next.GetAllEstablishments(ctx)
}

/*************** Main ***************/
/* Main */
func BuildGetAllEstablishmentsEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetAllEstablishments")
		csLogger := log.With(logger, "method", "GetAllEstablishments")

		csEndpoint = GetAllEstablishmentsEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetAllEstablishments")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
