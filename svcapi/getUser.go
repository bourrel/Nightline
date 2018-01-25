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
func (s Service) GetUser(ctx context.Context, userID int64) (svcdb.Profile, error) {
	user, err := s.svcdb.GetUserProfile(ctx, userID)
	return user, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type getUserRequest struct {
	UserID int64 `json:"id"`
}

type getUserResponse struct {
	User svcdb.Profile `json:"user"`
}

func GetUserEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getUserRequest)
		user, err := svc.GetUser(ctx, req.UserID)
		if err != nil {
			fmt.Println("Error GetUserEndpoint : ", err.Error())
			return getUserResponse{user}, err
		}
		return getUserResponse{user}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetUserRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getUserRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPGetUserRequest 1 : ", err.Error())
		return nil, err
	}

	userID, err := strconv.ParseInt(mux.Vars(r)["UserId"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPGetUserRequest 2 : ", err.Error())
		return nil, RequestError
	}
	(&request).UserID = userID

	return request, nil
}

func DecodeHTTPGetUserResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getUserResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPGetUserResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func GetUserHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/users/{UserId:[0-9]+}").Handler(httptransport.NewServer(
		endpoints.GetUserEndpoint,
		DecodeHTTPGetUserRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetUser", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetUser(ctx context.Context, userID int64) (svcdb.Profile, error) {
	profile, err := mw.next.GetUser(ctx, userID)

	mw.logger.Log(
		"method", "getUser",
		"userID", userID,
		"response", profile,
		"took", time.Since(time.Now()),
	)
	return profile, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) GetUser(ctx context.Context, userID int64) (svcdb.Profile, error) {
	return mw.next.GetUser(ctx, userID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetUser(ctx context.Context, userID int64) (svcdb.Profile, error) {
	return mw.next.GetUser(ctx, userID)
}

/*************** Main ***************/
/* Main */
func BuildGetUserEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetUser")
		csLogger := log.With(logger, "method", "GetUser")

		csEndpoint = GetUserEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetUser")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
