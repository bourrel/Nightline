package svcdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
func (s Service) GetUsersInvitations(_ context.Context, userID int64) ([]Invitation, error) {
	var invitations []Invitation

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetUsersInvitations (WaitConnection) : " + err.Error())
		return nil, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo("MATCH (u:USER)<-[i:INVITE]-(f:USER) WHERE ID(u) = {id} RETURN f, i, u")
	if err != nil {
		fmt.Println("GetUsersInvitations (PrepareNeo)")
		panic(err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": userID,
	})
	if err != nil {
		fmt.Println("GetUsersInvitations (QueryNeo)")
		panic(err)
	}

	// Here we loop through the rows until we get the metadata object
	// back, meaning the row stream has been fully consumed
	var tmpInvitation Invitation

	row, _, err := rows.NextNeo()
	for row != nil && err == nil {

		if err != nil && err != io.EOF {
			fmt.Println("GetUsersInvitations (---)")
			panic(err)
		} else if err != io.EOF {
			(&tmpInvitation).RelationToInvitation(
				row[0].(graph.Node),
				row[1].(graph.Relationship),
				row[2].(graph.Node))

			invitations = append(invitations, tmpInvitation)
		}
		row, _, err = rows.NextNeo()
	}
	return invitations, nil
}

/*************** Endpoint ***************/
type getUsersInvitationsByEstablishmentRequest struct {
	UserID int64 `json:"id"`
}

type getUsersInvitationsByEstablishmentResponse struct {
	Invitations []Invitation `json:"invitations"`
	Err         string       `json:"err,omitempty"`
}

func GetUsersInvitationsEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getUsersInvitationsByEstablishmentRequest)
		invitations, err := svc.GetUsersInvitations(ctx, req.UserID)
		if err != nil {
			fmt.Println("Error GetUsersInvitationsEndpoint : ", err.Error())
			return getUsersInvitationsByEstablishmentResponse{invitations, err.Error()}, nil
		}
		return getUsersInvitationsByEstablishmentResponse{invitations, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetUsersInvitationsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getUsersInvitationsByEstablishmentRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPGetUsersInvitationsRequest 1 : ", err.Error())
		return nil, err
	}

	userID, err := strconv.ParseInt(mux.Vars(r)["UserID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPGetUsersInvitationsRequest 2 : ", err.Error())
		return nil, err
	}
	(&request).UserID = userID

	return request, nil
}

func DecodeHTTPGetUsersInvitationsResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getUsersInvitationsByEstablishmentResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPGetUsersInvitationsResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func GetUsersInvitationsHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").
		Path("/users/{UserID}/invitations/users").
		Handler(httptransport.NewServer(
			endpoints.GetUsersInvitationsEndpoint,
			DecodeHTTPGetUsersInvitationsRequest,
			EncodeHTTPGenericResponse,
			append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetUsersInvitations", logger)))...,
		))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetUsersInvitations(ctx context.Context, userID int64) ([]Invitation, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getUsersInvitationsByEstablishment",
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetUsersInvitations(ctx, userID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetUsersInvitations(ctx context.Context, userID int64) ([]Invitation, error) {
	v, err := mw.next.GetUsersInvitations(ctx, userID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetUsersInvitationsEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetUsersInvitations")
		csLogger := log.With(logger, "method", "GetUsersInvitations")

		csEndpoint = GetUsersInvitationsEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetUsersInvitations")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetUsersInvitations(ctx context.Context, userID int64) ([]Invitation, error) {
	var et []Invitation

	request := getUsersInvitationsByEstablishmentRequest{UserID: userID}
	response, err := e.GetUsersInvitationsEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error GetUsersInvitations : ", err.Error())
		return et, err
	}
	et = response.(getUsersInvitationsByEstablishmentResponse).Invitations
	return et, str2err(response.(getUsersInvitationsByEstablishmentResponse).Err)
}

func EncodeHTTPGetUsersInvitationsRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(getUsersInvitationsByEstablishmentRequest).UserID)
	encodedUrl, err := route.Path(r.URL.Path).URL("UserID", id)
	if err != nil {
		fmt.Println("Error EncodeHTTPGetUsersInvitationsRequest : ", err.Error())
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetUsersInvitations(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/users/{UserID}/invitations/users"),
		EncodeHTTPGetUsersInvitationsRequest,
		DecodeHTTPGetUsersInvitationsResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "GetUsersInvitations")(ceEndpoint)
	return ceEndpoint, nil
}
