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
func (s Service) InvitationDecline(ctx context.Context, invitationID int64) error {
	_, _, err := s.svcdb.InvitationDecline(ctx, invitationID)

	return dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type invitationDeclineRequest struct {
	InvitationID int64 `json:"invitationID"`
}

type invitationDeclineResponse struct {
}

func InvitationDeclineEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(invitationDeclineRequest)
		err := svc.InvitationDecline(ctx, req.InvitationID)
		return invitationDeclineResponse{}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPInvitationDeclineRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request invitationDeclineRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPInvitationDeclineRequest 1 : ", err.Error())
		return nil, err
	}

	invitationID, err := strconv.ParseInt(mux.Vars(r)["InvitationID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPInvitationDeclineRequest 3 : ", err.Error())
		return nil, RequestError
	}
	(&request).InvitationID = invitationID

	return request, nil
}

func DecodeHTTPInvitationDeclineResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response invitationDeclineResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPInvitationDeclineResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func InvitationDeclineHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/invitations/users/{InvitationID:[0-9]+}/decline").Handler(httptransport.NewServer(
		endpoints.InvitationDeclineEndpoint,
		DecodeHTTPInvitationDeclineRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "InvitationDecline", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) InvitationDecline(ctx context.Context, invitationID int64) error {
	err := mw.next.InvitationDecline(ctx, invitationID)

	mw.logger.Log(
		"method", "InvitationDecline",
		"invitationID", invitationID,
		"took", time.Since(time.Now()),
	)
	return err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) InvitationDecline(ctx context.Context, invitationID int64) error {
	return mw.next.InvitationDecline(ctx, invitationID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) InvitationDecline(ctx context.Context, invitationID int64) error {
	return mw.next.InvitationDecline(ctx, invitationID)
}

/*************** Main ***************/
/* Main */
func BuildInvitationDeclineEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var gefmEndpoint endpoint.Endpoint
	{
		gefmDuration := duration.With("method", "InvitationDecline")
		gefmLogger := log.With(logger, "method", "InvitationDecline")

		gefmEndpoint = InvitationDeclineEndpoint(svc)
		gefmEndpoint = opentracing.TraceServer(tracer, "InvitationDecline")(gefmEndpoint)
		gefmEndpoint = EndpointLoggingMiddleware(gefmLogger)(gefmEndpoint)
		gefmEndpoint = EndpointAuthenticationMiddleware()(gefmEndpoint)
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}
