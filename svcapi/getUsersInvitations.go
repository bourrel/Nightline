package svcapi

import (
	"context"
	"encoding/json"
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
func (s Service) GetUsersInvitations(ctx context.Context, userID int64) ([]svcdb.Invitation, error) {
	invitations, err := s.svcdb.GetUsersInvitations(ctx, userID)
	return invitations, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type getUsersInvitationsRequest struct {
	UserID int64 `json:"userID"`
}

type getUsersInvitationsResponse struct {
	Invitations []svcdb.Invitation `json:"invitations"`
}

func GetUsersInvitationsEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {

		req := request.(getUsersInvitationsRequest)
		invitations, err := svc.GetUsersInvitations(ctx, req.UserID)
		if err != nil {
			fmt.Println("Error GetUsersInvitationsEndpoint : ", err.Error())
			return getUsersInvitationsResponse{invitations}, err
		}
		return getUsersInvitationsResponse{invitations}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPgetUsersInvitationsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getUsersInvitationsRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPgetUsersInvitationsRequest 1 : ", err.Error())
		return nil, err
	}

	userID, err := strconv.ParseInt(mux.Vars(r)["UserID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPgetUsersInvitationsRequest 2 : ", err.Error())
		return nil, RequestError
	}
	(&request).UserID = userID

	return request, nil
}

func DecodeHTTPgetUsersInvitationsResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getUsersInvitationsResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, RequestError
	}
	return response, nil
}

func GetUsersInvitationsHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/users/{UserID:[0-9]+}/invitations/users").Handler(httptransport.NewServer(
		endpoints.GetUsersInvitationsEndpoint,
		DecodeHTTPgetUsersInvitationsRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetUsersInvitations", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetUsersInvitations(ctx context.Context, userID int64) ([]svcdb.Invitation, error) {
	estabs, err := mw.next.GetUsersInvitations(ctx, userID)

	mw.logger.Log(
		"method", "GetUsersInvitations",
		"response", estabs,
		"took", time.Since(time.Now()),
	)
	return estabs, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) GetUsersInvitations(ctx context.Context, userID int64) ([]svcdb.Invitation, error) {
	return mw.next.GetUsersInvitations(ctx, userID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetUsersInvitations(ctx context.Context, userID int64) ([]svcdb.Invitation, error) {
	return mw.next.GetUsersInvitations(ctx, userID)
}

/*************** Main ***************/
/* Main */
func BuildGetUsersInvitationsEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetUsersInvitations")
		csLogger := log.With(logger, "method", "GetUsersInvitations")

		csEndpoint = GetUsersInvitationsEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetUsersInvitations")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
