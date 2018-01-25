package svcapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"svcws"
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
func (s Service) GroupInvite(ctx context.Context, groupID, friendID int64) error {
	name, invitationID, err := s.svcdb.GroupInvite(ctx, groupID, friendID)

	notif := &groupInviteNotif{
		GroupID:      groupID,
		Name:         name,
		InvitationID: invitationID,
	}

	if err == nil {
		s.svcevent.Push(ctx, svcws.GroupInvitation, notif, friendID)
	}

	return dbToHTTPErr(err)
}

/*************** Endpoint ***************/

type groupInviteRequest struct {
	GroupID  int64 `json:"groupID"`
	FriendID int64 `json:"friendID"`
}

type groupInviteResponse struct {
}

func GroupInviteEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupInviteRequest)
		err := svc.GroupInvite(ctx, req.GroupID, req.FriendID)
		return groupInviteResponse{}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPGroupInviteRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request groupInviteRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPGroupInviteRequest 1 : ", err.Error())
		return nil, err
	}

	groupID, err := strconv.ParseInt(mux.Vars(r)["GroupID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPGroupInviteRequest 2 : ", err.Error())
		return nil, RequestError
	}
	friendID, err := strconv.ParseInt(mux.Vars(r)["FriendID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPGroupInviteRequest 3 : ", err.Error())
		return nil, RequestError
	}
	(&request).FriendID = friendID
	(&request).GroupID = groupID

	return request, nil
}

func DecodeHTTPGroupInviteResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response groupInviteResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPGroupInviteResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func GroupInviteHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/groups/{GroupID:[0-9]+}/invite/{FriendID:[0-9]+}").Handler(httptransport.NewServer(
		endpoints.GroupInviteEndpoint,
		DecodeHTTPGroupInviteRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GroupInvite", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GroupInvite(ctx context.Context, groupID, friendID int64) error {
	err := mw.next.GroupInvite(ctx, groupID, friendID)

	mw.logger.Log(
		"method", "GroupInvite",
		"groupID", groupID,
		"friendID", friendID,
		"took", time.Since(time.Now()),
	)
	return err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) GroupInvite(ctx context.Context, groupID, friendID int64) error {
	return mw.next.GroupInvite(ctx, groupID, friendID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GroupInvite(ctx context.Context, groupID, friendID int64) error {
	return mw.next.GroupInvite(ctx, groupID, friendID)
}

/*************** Main ***************/
/* Main */
func BuildGroupInviteEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var gefmEndpoint endpoint.Endpoint
	{
		gefmDuration := duration.With("method", "GroupInvite")
		gefmLogger := log.With(logger, "method", "GroupInvite")

		gefmEndpoint = GroupInviteEndpoint(svc)
		gefmEndpoint = opentracing.TraceServer(tracer, "GroupInvite")(gefmEndpoint)
		gefmEndpoint = EndpointLoggingMiddleware(gefmLogger)(gefmEndpoint)
		gefmEndpoint = EndpointAuthenticationMiddleware()(gefmEndpoint)
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}
