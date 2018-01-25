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
func (s Service) GroupInvitationDecline(_ context.Context, _, invitationID int64) error {

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Fprintf(os.Stdout, "GroupInvitationDecline (WaitConnection) : "+err.Error())
		return err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (u:USER)<-[i:INVITE]-(g:GROUP) WHERE ID(i) = {invitationID} DELETE i
	`)
	defer stmt.Close()
	if err != nil {
		fmt.Println("Error GroupInvitationDecline (PrepareNeo) : " + err.Error())
		return err
	}

	_, err = stmt.QueryNeo(map[string]interface{}{
		"invitationID": invitationID,
	})

	if err != nil {
		fmt.Println("Error GroupInvitationDecline (QueryNeo) : " + err.Error())
		return err
	}

	return nil
}

/*************** Endpoint ***************/
type groupInvitationDeclineRequest struct {
	GroupID      int64 `json:"userID"`
	InvitationID int64 `json:"invitationID"`
}

type groupInvitationDeclineResponse struct {
	Err string `json:"err,omitempty"`
}

func GroupInvitationDeclineEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupInvitationDeclineRequest)
		err := svc.GroupInvitationDecline(ctx, req.GroupID, req.InvitationID)
		if err != nil {
			fmt.Println("Error GroupInvitationDeclineEndpoint : ", err.Error())
			return groupInvitationDeclineResponse{Err: err.Error()}, nil
		}
		return nil, nil
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
		fmt.Println("Error DecodeHTTPGroupInvitationDeclineRequest 2 : ", err.Error())
		return nil, err
	}
	(&request).InvitationID = invitationID

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
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GroupInvitationDecline", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GroupInvitationDecline(ctx context.Context, userID, invitationID int64) error {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "groupInvitationDecline",
			"userID", userID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GroupInvitationDecline(ctx, userID, invitationID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GroupInvitationDecline(ctx context.Context, userID, invitationID int64) error {
	err := mw.next.GroupInvitationDecline(ctx, userID, invitationID)
	mw.ints.Add(1)
	return err
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
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GroupInvitationDecline(ctx context.Context, userID, invitationID int64) error {
	request := groupInvitationDeclineRequest{GroupID: userID, InvitationID: invitationID}
	response, err := e.GroupInvitationDeclineEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error GroupInvitationDecline : ", err.Error())
		return err
	}
	return str2err(response.(groupInvitationDeclineResponse).Err)
}

func EncodeHTTPGroupInvitationDeclineRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	fid := strconv.FormatInt(request.(groupInvitationDeclineRequest).InvitationID, 10)

	encodedUrl, err := route.Path(r.URL.Path).URL("InvitationID", fid)
	if err != nil {
		fmt.Println("Error EncodeHTTPGroupInvitationDeclineRequest : ", err.Error())
		return err
	}

	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGroupInvitationDecline(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/invitations/groups/{InvitationID:[0-9]+}/decline"),
		EncodeHTTPGroupInvitationDeclineRequest,
		DecodeHTTPGroupInvitationDeclineResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "GroupInvitationDecline")(gefmEndpoint)
	return gefmEndpoint, nil
}
