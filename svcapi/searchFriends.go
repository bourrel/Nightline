package svcapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
func (s Service) SearchFriends(ctx context.Context, query string, userID int64) ([]svcdb.SearchResponse, error) {
	responses, err := s.svcdb.SearchFriends(ctx, query, userID)
	return responses, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type SearchFriendsRequest struct {
	query  string `json:"query"`
	userID int64  `json:"user"`
}

type SearchFriendsResponse struct {
	User []svcdb.SearchResponse `json:"Users"`
}

func SearchFriendsEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(SearchFriendsRequest)
		User, err := svc.SearchFriends(ctx, req.query, req.userID)
		return SearchFriendsResponse{User: User}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPSearchFriendsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req SearchFriendsRequest

	if len(r.URL.Query()) <= 0 {
		err := errors.New("Empty query")
		fmt.Println("Error DecodeHTTPSearchFriendsRequest : ", err.Error())
		return nil, err
	}

	(&req).query = r.URL.Query()["q"][0]
	if (&req).query == "" {
		err := errors.New("Invalid query, missing q params")
		fmt.Println("Error DecodeHTTPSearchFriendsRequest 1 : ", err.Error())
		return nil, err
	}

	userID := r.URL.Query()["userID"]
	if len(userID) < 1 || userID[0] == "" {
		err := errors.New("Invalid query, missing userID params")
		fmt.Println("Error DecodeHTTPSearchFriendsRequest 2 : ", err.Error())
		return nil, err
	}

	tmpUserID, err := strconv.ParseInt(userID[0], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPSearchFriendsRequest 3 : ", err.Error())
		return nil, err
	}
	req.userID = tmpUserID

	return req, nil
}

func DecodeHTTPSearchFriendsResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response SearchFriendsResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func SearchFriendsHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/search/friends").Handler(httptransport.NewServer(
		endpoints.SearchFriendsEndpoint,
		DecodeHTTPSearchFriendsRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "SearchFriends", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) SearchFriends(ctx context.Context, query string, userID int64) ([]svcdb.SearchResponse, error) {
	users, err := mw.next.SearchFriends(ctx, query, userID)

	mw.logger.Log(
		"method", "SearchFriends",
		"query", query,
		"user", userID,
		"took", time.Since(time.Now()),
	)
	return users, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) SearchFriends(ctx context.Context, query string, userID int64) ([]svcdb.SearchResponse, error) {
	return mw.next.SearchFriends(ctx, query, userID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) SearchFriends(ctx context.Context, query string, userID int64) ([]svcdb.SearchResponse, error) {
	return mw.next.SearchFriends(ctx, query, userID)
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
