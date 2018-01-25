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
func (s Service) InvitationAccept(_ context.Context, invitationID int64) (Profile, Profile, error) {
	var user Profile
	var friend Profile

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("InvitationAccept (WaitConnection) : " + err.Error())
		return user, friend, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (u:USER)<-[i:INVITE]-(f:USER)
		WHERE ID(i) = {invitationID}
		DELETE i
		CREATE (u)-[:KNOW]->(f)
		RETURN u, f
	`)
	defer stmt.Close()
	if err != nil {
		fmt.Println("Error InvitationAccept (PrepareNeo) : " + err.Error())
		return user, friend, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"invitationID": invitationID,
	})

	if err != nil {
		fmt.Println("Error InvitationAccept (QueryNeo) : " + err.Error())
		return user, friend, err
	}

	data, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("Error InvitationAccept (NextNeo) : " + err.Error())
		return user, friend, err
	}

	(&user).NodeToProfile(data[0].(graph.Node))
	(&friend).NodeToProfile(data[1].(graph.Node))

	return user, friend, nil
}

/*************** Endpoint ***************/
type invitationAcceptRequest struct {
	InvitationID int64 `json:"invitationID"`
}

type invitationAcceptResponse struct {
	User   Profile `json:"userID"`
	Friend Profile `json:"friendID"`
	Err    string  `json:"err,omitempty"`
}

func InvitationAcceptEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(invitationAcceptRequest)

		user, friend, err := svc.InvitationAccept(ctx, req.InvitationID)
		if err != nil {
			fmt.Println("Error InvitationAcceptEndpoint : ", err.Error())
			return invitationAcceptResponse{Err: err.Error()}, nil
		}
		return invitationAcceptResponse{User: user, Friend: friend}, nil
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
		fmt.Println("Error DecodeHTTPInvitationAcceptRequest 2 : ", err.Error())
		return nil, err
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
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "InvitationAccept", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) InvitationAccept(ctx context.Context, invitationID int64) (Profile, Profile, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "invitationAccept",
			"invitationID", invitationID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.InvitationAccept(ctx, invitationID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) InvitationAccept(ctx context.Context, invitationID int64) (Profile, Profile, error) {
	user, friend, err := mw.next.InvitationAccept(ctx, invitationID)
	mw.ints.Add(1)
	return user, friend, err
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
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) InvitationAccept(ctx context.Context, invitationID int64) (Profile, Profile, error) {
	request := invitationAcceptRequest{InvitationID: invitationID}
	response, err := e.InvitationAcceptEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error InvitationAccept : ", err.Error())
		return Profile{}, Profile{}, err
	}

	r := response.(invitationAcceptResponse)
	return r.User, r.Friend, str2err(r.Err)
}

func EncodeHTTPInvitationAcceptRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	fid := strconv.FormatInt(request.(invitationAcceptRequest).InvitationID, 10)

	encodedUrl, err := route.Path(r.URL.Path).URL("InvitationID", fid)
	if err != nil {
		fmt.Println("Error EncodeHTTPInvitationAcceptRequest : ", err.Error())
		return err
	}

	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientInvitationAccept(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/invitations/users/{InvitationID:[0-9]+}/accept"),
		EncodeHTTPInvitationAcceptRequest,
		DecodeHTTPInvitationAcceptResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "InvitationAccept")(gefmEndpoint)
	return gefmEndpoint, nil
}
