package svcdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-kit/kit/tracing/opentracing"
	mux "github.com/gorilla/mux"
	"github.com/johnnadratowski/golang-neo4j-bolt-driver/structures/graph"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"
)

/*************** Service ***************/
func (s Service) InvitationDecline(_ context.Context, invitationID int64) (Profile, Profile, error) {
	var user Profile
	var friend Profile

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("InvitationDecline (WaitConnection) : " + err.Error())
		return user, friend, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (u:USER)<-[i:INVITE]-(f:USER)
		WHERE ID(i) = {invitationID}
		DELETE i
		RETURN u, f
	`)
	defer stmt.Close()
	if err != nil {
		fmt.Println("Error InvitationDecline (PrepareNeo) : " + err.Error())
		return user, friend, err
	}
	rows, err := stmt.QueryNeo(map[string]interface{}{
		"invitationID": invitationID,
	})

	if err != nil {
		fmt.Println("Error InvitationDecline (QueryNeo) : " + err.Error())
		return user, friend, err
	}

	data, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("Error InvitationDecline (NextNeo) : " + err.Error())
		return user, friend, err
	}

	(&user).NodeToProfile(data[0].(graph.Node))
	(&friend).NodeToProfile(data[1].(graph.Node))

	return user, friend, nil
}

/*************** Endpoint ***************/
type invitationDeclineRequest struct {
	InvitationID int64 `json:"invitationID"`
}

type invitationDeclineResponse struct {
	User   Profile `json:"userID"`
	Friend Profile `json:"friendID"`
	Err    string  `json:"err,omitempty"`
}

func InvitationDeclineEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(invitationDeclineRequest)

		user, friend, err := svc.InvitationDecline(ctx, req.InvitationID)
		if err != nil {
			fmt.Println("Error InvitationDeclineEndpoint : ", err.Error())
			return invitationDeclineResponse{Err: err.Error()}, nil
		}
		return invitationDeclineResponse{User: user, Friend: friend}, nil
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
		fmt.Println("Error DecodeHTTPInvitationDeclineRequest 2 : ", err.Error())
		return nil, err
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
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "InvitationDecline", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) InvitationDecline(ctx context.Context, invitationID int64) (Profile, Profile, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "invitationDecline",
			"invitationID", invitationID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.InvitationDecline(ctx, invitationID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) InvitationDecline(ctx context.Context, invitationID int64) (Profile, Profile, error) {
	user, friend, err := mw.next.InvitationDecline(ctx, invitationID)
	mw.ints.Add(1)
	return user, friend, err
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
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) InvitationDecline(ctx context.Context, invitationID int64) (Profile, Profile, error) {
	request := invitationDeclineRequest{InvitationID: invitationID}
	response, err := e.InvitationDeclineEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error InvitationDecline : ", err.Error())
		return Profile{}, Profile{}, err
	}
	r := response.(invitationDeclineResponse)
	return r.User, r.Friend, str2err(r.Err)
}

func EncodeHTTPInvitationDeclineRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	fid := strconv.FormatInt(request.(invitationDeclineRequest).InvitationID, 10)

	encodedUrl, err := route.Path(r.URL.Path).URL("InvitationID", fid)
	if err != nil {
		fmt.Println("Error EncodeHTTPInvitationDeclineRequest : ", err.Error())
		return err
	}

	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientInvitationDecline(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/invitations/users/{InvitationID:[0-9]+}/decline"),
		EncodeHTTPInvitationDeclineRequest,
		DecodeHTTPInvitationDeclineResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "InvitationDecline")(gefmEndpoint)
	return gefmEndpoint, nil
}
