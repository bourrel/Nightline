package svcapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"svcdb"
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
func (s Service) GetUserFriends(ctx context.Context, userID int64) ([]svcdb.Profile, error) {
	friends, err := s.svcdb.GetUserFriends(ctx, userID)
	return friends, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type getUserFriendsRequest struct {
	UserID int64 `json:"userID"`
}

type getUserFriendsResponse struct {
	Profile []svcdb.Profile `json:"friends"`
}

func GetUserFriendsEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getUserFriendsRequest)
		menu, err := svc.GetUserFriends(ctx, req.UserID)
		return getUserFriendsResponse{Profile: menu}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPGetUserFriendsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getUserFriendsRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPGetUserFriendsRequest 1 : ", err.Error())
		return nil, err
	}

	userID, err := strconv.ParseInt(mux.Vars(r)["UserID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPGetUserFriendsRequest 2 : ", err.Error())
		return nil, RequestError
	}
	(&request).UserID = userID

	return request, nil
}

func DecodeHTTPGetUserFriendsResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getUserFriendsResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPGetUserFriendsResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func GetUserFriendsHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/users/{UserID:[0-9]+}/friends").Handler(httptransport.NewServer(
		endpoints.GetUserFriendsEndpoint,
		DecodeHTTPGetUserFriendsRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetUserFriends", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetUserFriends(ctx context.Context, userID int64) ([]svcdb.Profile, error) {
	menu, err := mw.next.GetUserFriends(ctx, userID)

	mw.logger.Log(
		"method", "getUserFriends",
		"userID", userID,
		"response", menu,
		"took", time.Since(time.Now()),
	)
	return menu, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) GetUserFriends(ctx context.Context, userID int64) ([]svcdb.Profile, error) {
	return mw.next.GetUserFriends(ctx, userID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetUserFriends(ctx context.Context, userID int64) ([]svcdb.Profile, error) {
	return mw.next.GetUserFriends(ctx, userID)
}

/*************** Main ***************/
/* Main */
func BuildGetUserFriendsEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var gefmEndpoint endpoint.Endpoint
	{
		gefmDuration := duration.With("method", "GetUserFriends")
		gefmLogger := log.With(logger, "method", "GetUserFriends")

		gefmEndpoint = GetUserFriendsEndpoint(svc)
		gefmEndpoint = opentracing.TraceServer(tracer, "GetUserFriends")(gefmEndpoint)
		gefmEndpoint = EndpointLoggingMiddleware(gefmLogger)(gefmEndpoint)
		gefmEndpoint = EndpointAuthenticationMiddleware()(gefmEndpoint)
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}
