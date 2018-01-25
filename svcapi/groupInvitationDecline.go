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
func (s Service) GroupInvitationDecline(ctx context.Context, userID, invitationID int64) error {
	err := s.svcdb.GroupInvitationDecline(ctx, userID, invitationID)
	return dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type groupInvitationDeclineRequest struct {
	UserID       int64 `json:"userID"`
	InvitationID int64 `json:"invitationID"`
}

type groupInvitationDeclineResponse struct {
}

func GroupInvitationDeclineEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupInvitationDeclineRequest)
		err := svc.GroupInvitationDecline(ctx, req.UserID, req.InvitationID)
		return groupInvitationDeclineResponse{}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPGroupInvitationDeclineRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request groupInvitationDeclineRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPGroupInvitationDeclineRequest 1 : ", err.Error())
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

func DecodeHTTPGroupInvitationDeclineResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response groupInvitationDeclineResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPGroupInvitationDeclineResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func GroupInvitationDeclineHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/invitations/groups/{InvitationID:[0-9]+}/decline").Handler(httptransport.NewServer(
		endpoints.GroupInvitationDeclineEndpoint,
		DecodeHTTPGroupInvitationDeclineRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GroupInvitationDecline", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GroupInvitationDecline(ctx context.Context, userID, friendID int64) error {
	err := mw.next.GroupInvitationDecline(ctx, userID, friendID)

	mw.logger.Log(
		"method", "GroupInvitationDecline",
		"userID", userID,
		"friendID", friendID,
		"took", time.Since(time.Now()),
	)
	return err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) GroupInvitationDecline(ctx context.Context, userID, friendID int64) error {
	return mw.next.GroupInvitationDecline(ctx, userID, friendID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GroupInvitationDecline(ctx context.Context, userID, friendID int64) error {
	return mw.next.GroupInvitationDecline(ctx, userID, friendID)
}

/*************** Main ***************/
/* Main */
func BuildGroupInvitationDeclineEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var gefmEndpoint endpoint.Endpoint
	{
		gefmDuration := duration.With("method", "GroupInvitationDecline")
		gefmLogger := log.With(logger, "method", "GroupInvitationDecline")

		gefmEndpoint = GroupInvitationDeclineEndpoint(svc)
		gefmEndpoint = opentracing.TraceServer(tracer, "GroupInvitationDecline")(gefmEndpoint)
		gefmEndpoint = EndpointLoggingMiddleware(gefmLogger)(gefmEndpoint)
		gefmEndpoint = EndpointAuthenticationMiddleware()(gefmEndpoint)
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}
