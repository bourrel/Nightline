package svcdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/go-kit/kit/tracing/opentracing"
	mux "github.com/gorilla/mux"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"
)

/*************** Service ***************/
func (s Service) GroupInvitationAccept(_ context.Context, _, invitationID int64) error {

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Fprintf(os.Stdout, "GroupInvitationAccept (WaitConnection) : "+err.Error())
		return err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (u:USER)<-[i:INVITE]-(g:GROUP) WHERE ID(i) = {invitationID} DELETE i
		CREATE (u)-[m:MEMBER]->(g) RETURN u, g
	`)
	defer stmt.Close()
	if err != nil {
		fmt.Println("Error GroupInvitationAccept (PrepareNeo) : " + err.Error())
		return err
	}

	_, err = stmt.QueryNeo(map[string]interface{}{
		"invitationID": invitationID,
	})

	if err != nil {
		fmt.Println("Error GroupInvitationAccept (QueryNeo) : " + err.Error())
		return err
	}

	return nil
}

/*************** Endpoint ***************/
type groupGroupInvitationAcceptRequest struct {
	GroupID      int64 `json:"userID"`
	InvitationID int64 `json:"invitationID"`
}

type groupGroupInvitationAcceptResponse struct {
	Err string `json:"err,omitempty"`
}

func GroupInvitationAcceptEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupGroupInvitationAcceptRequest)
		err := svc.GroupInvitationAccept(ctx, req.GroupID, req.InvitationID)
		if err != nil {
			fmt.Println("Error GroupInvitationAcceptEndpoint : ", err.Error())
			return groupGroupInvitationAcceptResponse{Err: err.Error()}, nil
		}
		return nil, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGroupInvitationAcceptRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request groupGroupInvitationAcceptRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPGroupInvitationAcceptRequest 1 : ", err.Error())
		return nil, err
	}

	invitationID, err := strconv.ParseInt(mux.Vars(r)["InvitationID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPGroupInvitationAcceptRequest 2 : ", err.Error())
		return nil, err
	}
	(&request).InvitationID = invitationID

	return request, nil
}

func DecodeHTTPGroupInvitationAcceptResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response groupGroupInvitationAcceptResponse
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
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GroupInvitationAccept", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GroupInvitationAccept(ctx context.Context, userID, invitationID int64) error {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "groupGroupInvitationAccept",
			"userID", userID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GroupInvitationAccept(ctx, userID, invitationID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GroupInvitationAccept(ctx context.Context, userID, invitationID int64) error {
	err := mw.next.GroupInvitationAccept(ctx, userID, invitationID)
	mw.ints.Add(1)
	return err
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
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GroupInvitationAccept(ctx context.Context, userID, invitationID int64) error {
	request := groupGroupInvitationAcceptRequest{GroupID: userID, InvitationID: invitationID}
	response, err := e.GroupInvitationAcceptEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error GroupInvitationAccept : ", err.Error())
		return err
	}
	return str2err(response.(groupGroupInvitationAcceptResponse).Err)
}

func EncodeHTTPGroupInvitationAcceptRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	fid := strconv.FormatInt(request.(groupGroupInvitationAcceptRequest).InvitationID, 10)

	encodedUrl, err := route.Path(r.URL.Path).URL("InvitationID", fid)
	if err != nil {
		fmt.Println("Error EncodeHTTPGroupInvitationAcceptRequest : ", err.Error())
		return err
	}

	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGroupInvitationAccept(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/invitations/groups/{InvitationID:[0-9]+}/accept"),
		EncodeHTTPGroupInvitationAcceptRequest,
		DecodeHTTPGroupInvitationAcceptResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "GroupInvitationAccept")(gefmEndpoint)
	return gefmEndpoint, nil
}
