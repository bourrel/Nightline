package svcapi

import (
	"context"
	"encoding/json"
	"errors"
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
func (s Service) SearchEstablishments(ctx context.Context, query string) ([]svcdb.SearchResponse, error) {
	responses, err := s.svcdb.SearchEstablishments(ctx, query)
	return responses, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type SearchEstablishmentsRequest struct {
	query string `json:"query"`
}

type SearchEstablishmentsResponse struct {
	Establishment []svcdb.SearchResponse `json:"establishments"`
}

func SearchEstablishmentsEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(SearchEstablishmentsRequest)
		Establishment, err := svc.SearchEstablishments(ctx, req.query)
		return SearchEstablishmentsResponse{Establishment: Establishment}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPSearchEstablishmentsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req SearchEstablishmentsRequest

	if len(r.URL.Query()) <= 0 {
		err := errors.New("Empty query")
		fmt.Println("Error DecodeHTTPSearchEstablishmentsRequest : ", err.Error())
		return nil, err
	}

	(&req).query = r.URL.Query()["q"][0]

	if (&req).query == "" {
		err := errors.New("Invalid query")
		fmt.Println("Error DecodeHTTPSearchEstablishmentsRequest : ", err.Error())
		return nil, err
	}

	return req, nil
}

func DecodeHTTPSearchEstablishmentsResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response SearchEstablishmentsResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error on DecodeHTTPSearchEstablishmentsResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func SearchEstablishmentsHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/search/establishments").Handler(httptransport.NewServer(
		endpoints.SearchEstablishmentsEndpoint,
		DecodeHTTPSearchEstablishmentsRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "SearchEstablishments", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) SearchEstablishments(ctx context.Context, query string) ([]svcdb.SearchResponse, error) {
	estabs, err := mw.next.SearchEstablishments(ctx, query)

	mw.logger.Log(
		"method", "SearchEstablishments",
		"query", query,
		"response", estabs,
		"took", time.Since(time.Now()),
	)
	return estabs, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) SearchEstablishments(ctx context.Context, query string) ([]svcdb.SearchResponse, error) {
	return mw.next.SearchEstablishments(ctx, query)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) SearchEstablishments(ctx context.Context, query string) ([]svcdb.SearchResponse, error) {
	return mw.next.SearchEstablishments(ctx, query)
}

/*************** Main ***************/
/* Main */
func BuildSearchEstablishmentsEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "SearchEstablishments")
		csLogger := log.With(logger, "method", "SearchEstablishments")

		csEndpoint = SearchEstablishmentsEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "SearchEstablishments")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
