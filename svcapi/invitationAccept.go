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

func firstFriendSuccess(s Service, ctx context.Context, userID int64) {
	// Check if user has already this success
	userSuccess, err := s.svcdb.GetUserSuccess(ctx, userID)
	if err != nil {
		return
	}

	for _, success := range userSuccess {
		if success.Active == true && success.Value == "first_friend" {
			return
		}
	}

	// Get user friends count
	userFriends, err := s.svcdb.GetUserFriends(ctx, userID)
	if err != nil || len(userFriends) != 1 {
		fmt.Println(err.Error())
		return
	}

	// Check if user has already this success
	newSuccess, err := s.svcdb.GetSuccessByValue(ctx, "first_friend")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// Create and send success notification
	notif := &successNotif{
		Name:      newSuccess.Name,
		SuccessID: newSuccess.ID,
	}
	s.svcevent.Push(ctx, svcws.Success, notif, userID)

	err = s.svcdb.AddSuccess(ctx, userID, "first_friend")
	if err != nil || len(userFriends) != 1 {
		fmt.Println(err.Error())
		return
	}
}

/*************** Service ***************/
func (s Service) InvitationAccept(ctx context.Context, invitationID int64) error {
	user, friend, err := s.svcdb.InvitationAccept(ctx, invitationID)
	if err != nil {
		return dbToHTTPErr(err)
	}

	// Send Invitation Accept Notification
	notif := &invitationAcceptNotif{
		Name:         user.Pseudo,
		UserID:       user.ID,
		InvitationID: invitationID,
		Accepted:     true,
	}
	s.svcevent.Push(ctx, svcws.UserInvitationAnswr, notif, friend.ID)

	// Send first friend success
	firstFriendSuccess(s, ctx, user.ID)
	firstFriendSuccess(s, ctx, friend.ID)

	return nil
}

/*************** Endpoint ***************/
type invitationAcceptRequest struct {
	InvitationID int64 `json:"invitationID"`
}

type invitationAcceptResponse struct {
}

func InvitationAcceptEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(invitationAcceptRequest)
		err := svc.InvitationAccept(ctx, req.InvitationID)
		return invitationAcceptResponse{}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPInvitationAcceptRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request invitationAcceptRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPInvitationAcceptRequest 1 : ", err.Error())
		return nil, err
	}

	invitationID, err := strconv.ParseInt(mux.Vars(r)["InvitationID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPInvitationAcceptRequest 3 : ", err.Error())
		return nil, RequestError
	}
	(&request).InvitationID = invitationID

	return request, nil
}

func DecodeHTTPInvitationAcceptResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response invitationAcceptResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPInvitationAcceptResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func InvitationAcceptHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/invitations/users/{InvitationID:[0-9]+}/accept").Handler(httptransport.NewServer(
		endpoints.InvitationAcceptEndpoint,
		DecodeHTTPInvitationAcceptRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "InvitationAccept", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) InvitationAccept(ctx context.Context, invitationID int64) error {
	err := mw.next.InvitationAccept(ctx, invitationID)

	mw.logger.Log(
		"method", "InvitationAccept",
		"invitationID", invitationID,
		"took", time.Since(time.Now()),
	)
	return err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) InvitationAccept(ctx context.Context, invitationID int64) error {
	return mw.next.InvitationAccept(ctx, invitationID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) InvitationAccept(ctx context.Context, invitationID int64) error {
	return mw.next.InvitationAccept(ctx, invitationID)
}

/*************** Main ***************/
/* Main */
func BuildInvitationAcceptEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var gefmEndpoint endpoint.Endpoint
	{
		gefmDuration := duration.With("method", "InvitationAccept")
		gefmLogger := log.With(logger, "method", "InvitationAccept")

		gefmEndpoint = InvitationAcceptEndpoint(svc)
		gefmEndpoint = opentracing.TraceServer(tracer, "InvitationAccept")(gefmEndpoint)
		gefmEndpoint = EndpointLoggingMiddleware(gefmLogger)(gefmEndpoint)
		gefmEndpoint = EndpointAuthenticationMiddleware()(gefmEndpoint)
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}
