package svcdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
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
func (s Service) SearchFriends(_ context.Context, query string, userID int64) ([]SearchResponse, error) {
	var tmpUser User
	var tmpResponse SearchResponse
	var response []SearchResponse

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("SearchFriends (WaitConnection) : " + err.Error())
		return nil, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (u:USER)-[:KNOW]-(f:USER)
		WHERE ID(u) = {id} AND LOWER(f.Pseudo) CONTAINS LOWER({query})
		RETURN f
	`)
	defer stmt.Close()
	if err != nil {
		fmt.Println("SearchFriends (PrepareNeo)")
		return nil, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"query": query,
		"id":    userID,
	})

	if err != nil {
		fmt.Println("SearchFriends (QueryNeo)")
		return nil, err
	}

	row, _, err := rows.NextNeo()
	for row != nil && err == nil {

		if err != nil && err != io.EOF {
			fmt.Println("SearchFriends (---)")
			panic(err)
		} else if err != io.EOF {
			(&tmpUser).NodeToUser(row[0].(graph.Node))
			tmpResponse.FromUser(tmpUser)
			response = append(response, tmpResponse)
		}
		row, _, err = rows.NextNeo()
	}

	fmt.Println(response)
	return response, nil
}

/*************** Endpoint ***************/
type searchFriendsRequest struct {
	query  string `json:"query"`
	userID int64  `json:"user"`
}

type searchFriendsResponse struct {
	Friends []SearchResponse `json:"friends"`
	Err     string           `json:"err,omitempty"`
}

func SearchFriendsEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(searchFriendsRequest)
		friends, err := svc.SearchFriends(ctx, req.query, req.userID)
		if err != nil {
			fmt.Println("Error SearchFriendsEndpoint : " + err.Error())
			return searchFriendsResponse{friends, err.Error()}, nil
		}
		return searchFriendsResponse{friends, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPSearchFriendsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request searchFriendsRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	(&request).query = mux.Vars(r)["query"]
	if (&request).query == "" {
		err := errors.New("Invalid query")
		fmt.Println("Error DecodeHTTPSearchFriendsRequest : ", err.Error())
		return nil, err
	}

	userID := mux.Vars(r)["userID"]
	if userID == "" {
		err := errors.New("Invalid query, missing userID params")
		fmt.Println("Error DecodeHTTPSearchFriendsRequest 2 : ", err.Error())
		return nil, err
	}

	tmpUserID, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPSearchFriendsRequest 3 : ", err.Error())
		return nil, err
	}
	(&request).userID = tmpUserID

	return request, nil
}

func DecodeHTTPSearchFriendsResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response searchFriendsResponse

	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPSearchFriendsResponse : " + err.Error())
		return nil, err
	}
	return response, nil
}

func SearchFriendsHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/search/friends?q={query}&userID={userID}").Handler(httptransport.NewServer(
		endpoints.SearchFriendsEndpoint,
		DecodeHTTPSearchFriendsRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "SearchFriends", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) SearchFriends(ctx context.Context, query string, userID int64) ([]SearchResponse, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "searchFriends",
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.SearchFriends(ctx, query, userID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) SearchFriends(ctx context.Context, query string, userID int64) ([]SearchResponse, error) {
	v, err := mw.next.SearchFriends(ctx, query, userID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildSearchFriendsEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "SearchFriends")
		csLogger := log.With(logger, "method", "SearchFriends")

		csEndpoint = SearchFriendsEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "SearchFriends")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forsearch limiter & circuitbreaker for now kthx
func (e Endpoints) SearchFriends(ctx context.Context, query string, userID int64) ([]SearchResponse, error) {
	var et []SearchResponse

	request := searchFriendsRequest{query: query, userID: userID}
	response, err := e.SearchFriendsEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error SearchFriends : " + err.Error())
		return et, err
	}
	et = response.(searchFriendsResponse).Friends
	return et, str2err(response.(searchFriendsResponse).Err)
}

func EncodeHTTPSearchFriendsRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()

	query := fmt.Sprintf("%v", request.(searchFriendsRequest).query)
	uID := fmt.Sprintf("%v", request.(searchFriendsRequest).userID)
	encodedURL, err := route.Path(r.URL.Path).Queries("query", query, "userID", uID).URL("query", query, "userID", uID)
	if err != nil {
		fmt.Println("Error EncodeHTTPSearchFriendsRequest : ", err.Error())
		return err
	}
	r.URL.Path = encodedURL.Path
	return nil
}

func ClientSearchFriends(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/search/friends?q={query}&userID={userID}"),
		EncodeHTTPSearchFriendsRequest,
		DecodeHTTPSearchFriendsResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "SearchFriends")(ceEndpoint)
	return ceEndpoint, nil
}
