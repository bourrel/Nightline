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
func (s Service) SearchUsers(ctx context.Context, query string) ([]svcdb.SearchResponse, error) {
	responses, err := s.svcdb.SearchUsers(ctx, query)
	return responses, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type SearchUsersRequest struct {
	query string `json:"query"`
}

type SearchUsersResponse struct {
	User []svcdb.SearchResponse `json:"Users"`
}

func SearchUsersEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(SearchUsersRequest)
		User, err := svc.SearchUsers(ctx, req.query)
		return SearchUsersResponse{User: User}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPSearchUsersRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req SearchUsersRequest

	if len(r.URL.Query()) <= 0 {
		err := errors.New("Empty query")
		fmt.Println("Error DecodeHTTPSearchUsersRequest : ", err.Error())
		return nil, err
	}

	(&req).query = r.URL.Query()["q"][0]
	if (&req).query == "" {
		err := errors.New("Invalid query")
		fmt.Println("Error DecodeHTTPSearchUsersRequest : ", err.Error())
		return nil, err
	}

	return req, nil
}

func DecodeHTTPSearchUsersResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response SearchUsersResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func SearchUsersHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/search/users").Handler(httptransport.NewServer(
		endpoints.SearchUsersEndpoint,
		DecodeHTTPSearchUsersRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "SearchUsers", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) SearchUsers(ctx context.Context, query string) ([]svcdb.SearchResponse, error) {
	users, err := mw.next.SearchUsers(ctx, query)

	mw.logger.Log(
		"method", "SearchUsers",
		"query", query,
		"response", users,
		"took", time.Since(time.Now()),
	)
	return users, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) SearchUsers(ctx context.Context, query string) ([]svcdb.SearchResponse, error) {
	return mw.next.SearchUsers(ctx, query)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) SearchUsers(ctx context.Context, query string) ([]svcdb.SearchResponse, error) {
	return mw.next.SearchUsers(ctx, query)
}

/*************** Main ***************/
/* Main */
func BuildSearchUsersEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "SearchUsers")
		csLogger := log.With(logger, "method", "SearchUsers")

		csEndpoint = SearchUsersEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "SearchUsers")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
