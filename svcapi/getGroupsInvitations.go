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
func (s Service) GetGroupsInvitations(ctx context.Context, groupID int64) ([]svcdb.GroupInvitation, error) {
	invitations, err := s.svcdb.GetGroupsInvitations(ctx, groupID)
	return invitations, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type getGroupsInvitationsRequest struct {
	GroupID int64 `json:"groupID"`
}

type getGroupsInvitationsResponse struct {
	Invitations []svcdb.GroupInvitation `json:"invitations"`
}

func GetGroupsInvitationsEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getGroupsInvitationsRequest)
		invitations, err := svc.GetGroupsInvitations(ctx, req.GroupID)
		if err != nil {
			fmt.Println("Error GetGroupsInvitationsEndpoint : ", err.Error())
			return getGroupsInvitationsResponse{invitations}, err
		}
		return getGroupsInvitationsResponse{invitations}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPgetGroupsInvitationsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getGroupsInvitationsRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPgetGroupsInvitationsRequest 1 : ", err.Error())
		return nil, err
	}

	groupID, err := strconv.ParseInt(mux.Vars(r)["GroupID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPgetGroupsInvitationsRequest 2 : ", err.Error())
		return nil, RequestError
	}
	(&request).GroupID = groupID

	return request, nil
}

func DecodeHTTPgetGroupsInvitationsResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getGroupsInvitationsResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, RequestError
	}
	return response, nil
}

func GetGroupsInvitationsHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/users/{GroupID:[0-9]+}/invitations/groups").Handler(httptransport.NewServer(
		endpoints.GetGroupsInvitationsEndpoint,
		DecodeHTTPgetGroupsInvitationsRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetGroupsInvitations", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetGroupsInvitations(ctx context.Context, groupID int64) ([]svcdb.GroupInvitation, error) {
	estabs, err := mw.next.GetGroupsInvitations(ctx, groupID)

	mw.logger.Log(
		"method", "GetGroupsInvitations",
		"response", estabs,
		"took", time.Since(time.Now()),
	)
	return estabs, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) GetGroupsInvitations(ctx context.Context, groupID int64) ([]svcdb.GroupInvitation, error) {
	return mw.next.GetGroupsInvitations(ctx, groupID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetGroupsInvitations(ctx context.Context, groupID int64) ([]svcdb.GroupInvitation, error) {
	return mw.next.GetGroupsInvitations(ctx, groupID)
}

/*************** Main ***************/
/* Main */
func BuildGetGroupsInvitationsEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetGroupsInvitations")
		csLogger := log.With(logger, "method", "GetGroupsInvitations")

		csEndpoint = GetGroupsInvitationsEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetGroupsInvitations")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
