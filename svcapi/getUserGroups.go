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
func (s Service) GetUserGroups(ctx context.Context, userID int64) ([]svcdb.GroupArrayElement, error) {
	groups, err := s.svcdb.GetUserGroups(ctx, userID)
	return groups, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type GetUserGroupsRequest struct {
	UserID int64 `json:"id"`
}

type GetUserGroupsResponse struct {
	Groups []svcdb.GroupArrayElement `json:"groups"`
}

func GetUserGroupsEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetUserGroupsRequest)
		pref, err := svc.GetUserGroups(ctx, req.UserID)
		return GetUserGroupsResponse{pref}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPGetUserGroupsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request GetUserGroupsRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPGetUserGroupsRequest 1 : ", err.Error())
		return nil, err
	}

	userID, err := strconv.ParseInt(mux.Vars(r)["UserId"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPGetUserGroupsRequest 2 : ", err.Error())
		return nil, RequestError
	}
	(&request).UserID = userID

	return request, nil
}

func DecodeHTTPGetUserGroupsResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response GetUserGroupsResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPGetUserGroupsResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func GetUserGroupsHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/users/{UserId:[0-9]+}/groups").Handler(httptransport.NewServer(
		endpoints.GetUserGroupsEndpoint,
		DecodeHTTPGetUserGroupsRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetUserGroups", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetUserGroups(ctx context.Context, userID int64) ([]svcdb.GroupArrayElement, error) {
	groups, err := mw.next.GetUserGroups(ctx, userID)

	mw.logger.Log(
		"method", "GetUserGroups",
		"userID", userID,
		"response", groups,
		"took", time.Since(time.Now()),
	)
	return groups, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) GetUserGroups(ctx context.Context, userID int64) ([]svcdb.GroupArrayElement, error) {
	return mw.next.GetUserGroups(ctx, userID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetUserGroups(ctx context.Context, userID int64) ([]svcdb.GroupArrayElement, error) {
	return mw.next.GetUserGroups(ctx, userID)
}

/*************** Main ***************/
/* Main */
func BuildGetUserGroupsEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetUserGroups")
		csLogger := log.With(logger, "method", "GetUserGroups")

		csEndpoint = GetUserGroupsEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetUserGroups")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
