package svcdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/go-kit/kit/tracing/opentracing"
	mux "github.com/gorilla/mux"
	"github.com/johnnadratowski/golang-neo4j-bolt-driver/structures/graph"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"
)

/*************** Service ***************/
func (s Service) SearchUsers(_ context.Context, query string) ([]SearchResponse, error) {
	var tmpUser User
	var tmpResponse SearchResponse
	var response []SearchResponse

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("SearchUsers (WaitConnection) : " + err.Error())
		return nil, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo("MATCH(u:USER) WHERE LOWER(u.Pseudo) CONTAINS LOWER({query}) RETURN u")
	defer stmt.Close()
	if err != nil {
		fmt.Println("SearchUsers (PrepareNeo)")
		return nil, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"query": query,
	})

	if err != nil {
		fmt.Println("SearchUsers (QueryNeo)")
		return nil, err
	}

	row, _, err := rows.NextNeo()
	for row != nil && err == nil {

		if err != nil && err != io.EOF {
			fmt.Println("SearchUsers (---)")
			panic(err)
		} else if err != io.EOF {
			(&tmpUser).NodeToUser(row[0].(graph.Node))
			tmpResponse.FromUser(tmpUser)
			response = append(response, tmpResponse)
		}
		row, _, err = rows.NextNeo()
	}
	return response, nil
}

/*************** Endpoint ***************/
type searchUsersRequest struct {
	query string `json:"query"`
}

type searchUsersResponse struct {
	Users []SearchResponse `json:"users"`
	Err   string           `json:"err,omitempty"`
}

func SearchUsersEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(searchUsersRequest)
		users, err := svc.SearchUsers(ctx, req.query)
		if err != nil {
			fmt.Println("Error SearchUsersEndpoint : " + err.Error())
			return searchUsersResponse{users, err.Error()}, nil
		}
		return searchUsersResponse{users, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPSearchUsersRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request searchUsersRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	(&request).query = mux.Vars(r)["query"]

	if (&request).query == "" {
		err := errors.New("Invalid query")
		fmt.Println("Error DecodeHTTPSearchUsersRequest : ", err.Error())
		return nil, err
	}

	return request, nil
}

func DecodeHTTPSearchUsersResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response searchUsersResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPSearchUsersResponse : " + err.Error())
		return nil, err
	}
	return response, nil
}

func SearchUsersHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/search/users?q={query}").Handler(httptransport.NewServer(
		endpoints.SearchUsersEndpoint,
		DecodeHTTPSearchUsersRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "SearchUsers", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) SearchUsers(ctx context.Context, query string) ([]SearchResponse, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "searchUsers",
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.SearchUsers(ctx, query)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) SearchUsers(ctx context.Context, query string) ([]SearchResponse, error) {
	v, err := mw.next.SearchUsers(ctx, query)
	mw.ints.Add(1)
	return v, err
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

/*************** Client ***************/
/* Client */
// Forsearch limiter & circuitbreaker for now kthx
func (e Endpoints) SearchUsers(ctx context.Context, query string) ([]SearchResponse, error) {
	var et []SearchResponse

	request := searchUsersRequest{query: query}
	response, err := e.SearchUsersEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error SearchUsers : " + err.Error())
		return et, err
	}
	et = response.(searchUsersResponse).Users
	return et, str2err(response.(searchUsersResponse).Err)
}

func EncodeHTTPSearchUsersRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()

	query := fmt.Sprintf("%v", request.(searchUsersRequest).query)
	encodedURL, err := route.Path(r.URL.Path).Queries("query", query).URL("query", query)
	if err != nil {
		fmt.Println("Error EncodeHTTPSearchUsersRequest : ", err.Error())
		return err
	}
	r.URL.Path = encodedURL.Path
	return nil
}

func ClientSearchUsers(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/search/users?q={query}"),
		EncodeHTTPSearchUsersRequest,
		DecodeHTTPSearchUsersResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "SearchUsers")(ceEndpoint)
	return ceEndpoint, nil
}
