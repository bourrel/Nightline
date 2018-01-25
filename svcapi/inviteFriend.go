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
func (s Service) InviteFriend(ctx context.Context, userID, friendID int64) error {
	user, invitationID, _, err := s.svcdb.InviteFriend(ctx, userID, friendID)
	if err != nil {
		return dbToHTTPErr(err)
	}

	notif := &userInviteNotif{
		UserID:       userID,
		Name:         user,
		InvitationID: invitationID,
	}
	s.svcevent.Push(ctx, svcws.UserInvitation, notif, friendID)

	return nil
}

/*************** Endpoint ***************/
type inviteFriendRequest struct {
	UserID   int64 `json:"userID"`
	FriendID int64 `json:"friendID"`
}

type inviteFriendResponse struct {
}

func InviteFriendEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(inviteFriendRequest)
		err := svc.InviteFriend(ctx, req.UserID, req.FriendID)
		return inviteFriendResponse{}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPInviteFriendRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request inviteFriendRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPInviteFriendRequest 1 : ", err.Error())
		return nil, err
	}

	userID, err := strconv.ParseInt(mux.Vars(r)["UserID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPInviteFriendRequest 2 : ", err.Error())
		return nil, RequestError
	}
	friendID, err := strconv.ParseInt(mux.Vars(r)["FriendID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPInviteFriendRequest 3 : ", err.Error())
		return nil, RequestError
	}
	(&request).FriendID = friendID
	(&request).UserID = userID

	return request, nil
}

func DecodeHTTPInviteFriendResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response inviteFriendResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPInviteFriendResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func InviteFriendHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/users/{UserID:[0-9]+}/friends/{FriendID:[0-9]+}/invite").Handler(httptransport.NewServer(
		endpoints.InviteFriendEndpoint,
		DecodeHTTPInviteFriendRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "InviteFriend", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) InviteFriend(ctx context.Context, userID, friendID int64) error {
	err := mw.next.InviteFriend(ctx, userID, friendID)

	mw.logger.Log(
		"method", "InviteFriend",
		"userID", userID,
		"friendID", friendID,
		"took", time.Since(time.Now()),
	)
	return err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) InviteFriend(ctx context.Context, userID, friendID int64) error {
	return mw.next.InviteFriend(ctx, userID, friendID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) InviteFriend(ctx context.Context, userID, friendID int64) error {
	return mw.next.InviteFriend(ctx, userID, friendID)
}

/*************** Main ***************/
/* Main */
func BuildInviteFriendEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var gefmEndpoint endpoint.Endpoint
	{
		gefmDuration := duration.With("method", "InviteFriend")
		gefmLogger := log.With(logger, "method", "InviteFriend")

		gefmEndpoint = InviteFriendEndpoint(svc)
		gefmEndpoint = opentracing.TraceServer(tracer, "InviteFriend")(gefmEndpoint)
		gefmEndpoint = EndpointLoggingMiddleware(gefmLogger)(gefmEndpoint)
		gefmEndpoint = EndpointAuthenticationMiddleware()(gefmEndpoint)
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}
