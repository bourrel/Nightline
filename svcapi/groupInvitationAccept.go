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
)

/*************** Service ***************/
func (s Service) GroupInvitationAccept(ctx context.Context, userID, invitationID int64) error {
	err := s.svcdb.GroupInvitationAccept(ctx, userID, invitationID)
	return dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type groupInvitationAcceptRequest struct {
	UserID       int64 `json:"userID"`
	InvitationID int64 `json:"invitationID"`
}

type groupInvitationAcceptResponse struct {
}

func GroupInvitationAcceptEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupInvitationAcceptRequest)
		err := svc.GroupInvitationAccept(ctx, req.UserID, req.InvitationID)
		return groupInvitationAcceptResponse{}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPGroupInvitationAcceptRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request groupInvitationAcceptRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPGroupInvitationAcceptRequest 1 : ", err.Error())
		return nil, err
	}

	invitationID, err := strconv.ParseInt(mux.Vars(r)["InvitationID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPGroupInvitationDeclineRequest 3 : ", err.Error())
		return nil, RequestError
	}

	(&request).InvitationID = invitationID
	(&request).UserID = 0

	return request, nil
}

func DecodeHTTPGroupInvitationAcceptResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response groupInvitationAcceptResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPGroupInvitationAcceptResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func GroupInvitationAcceptHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/invitations/groups/{InvitationID:[0-9]+}/accept").Handler(httptransport.NewServer(
		endpoints.GroupInvitationAcceptEndpoint,
		DecodeHTTPGroupInvitationAcceptRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GroupInvitationAccept", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GroupInvitationAccept(ctx context.Context, userID, friendID int64) error {
	err := mw.next.GroupInvitationAccept(ctx, userID, friendID)

	mw.logger.Log(
		"method", "GroupInvitationAccept",
		"userID", userID,
		"friendID", friendID,
		"took", time.Since(time.Now()),
	)
	return err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) GroupInvitationAccept(ctx context.Context, userID, friendID int64) error {
	return mw.next.GroupInvitationAccept(ctx, userID, friendID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GroupInvitationAccept(ctx context.Context, userID, friendID int64) error {
	return mw.next.GroupInvitationAccept(ctx, userID, friendID)
}

/*************** Main ***************/
/* Main */
func BuildGroupInvitationAcceptEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var gefmEndpoint endpoint.Endpoint
	{
		gefmDuration := duration.With("method", "GroupInvitationAccept")
		gefmLogger := log.With(logger, "method", "GroupInvitationAccept")

		gefmEndpoint = GroupInvitationAcceptEndpoint(svc)
		gefmEndpoint = opentracing.TraceServer(tracer, "GroupInvitationAccept")(gefmEndpoint)
		gefmEndpoint = EndpointLoggingMiddleware(gefmLogger)(gefmEndpoint)
		gefmEndpoint = EndpointAuthenticationMiddleware()(gefmEndpoint)
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}
