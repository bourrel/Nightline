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
func (s Service) GetGroupsInvitations(_ context.Context, groupID int64) ([]GroupInvitation, error) {
	var invitations []GroupInvitation

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetGroupsInvitations (WaitConnection) : " + err.Error())
		return nil, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (u:USER)<-[i:INVITE]-(g:GROUP)
		WHERE ID(u) = {id}
		RETURN g, i, u
	`)
	if err != nil {
		fmt.Println("GetGroupsInvitations (PrepareNeo)")
		panic(err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": groupID,
	})
	if err != nil {
		fmt.Println("GetGroupsInvitations (QueryNeo)")
		panic(err)
	}

	// Here we loop through the rows until we get the metadata object
	// back, meaning the row stream has been fully consumed
	var tmpInvitation GroupInvitation

	row, _, err := rows.NextNeo()
	for row != nil && err == nil {

		if err != nil && err != io.EOF {
			fmt.Println("GetGroupsInvitations (---)")
			panic(err)
		} else if err != io.EOF {
			(&tmpInvitation).RelationToGroupInvitation(
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
type getGroupsInvitationsRequest struct {
	UserID int64 `json:"id"`
}

type getGroupsInvitationsResponse struct {
	Invitations []GroupInvitation `json:"invitations"`
	Err         string            `json:"err,omitempty"`
}

func GetGroupsInvitationsEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getGroupsInvitationsRequest)
		invitations, err := svc.GetGroupsInvitations(ctx, req.UserID)
		if err != nil {
			fmt.Println("Error GetGroupsInvitationsEndpoint : ", err.Error())
			return getGroupsInvitationsResponse{invitations, err.Error()}, nil
		}

		for idx, invitation := range invitations {
			// invitation.ID

			tmpInvitation, err := svc.GetGroup(ctx, invitation.From.ID)
			if err != nil {
				fmt.Println("Error GetGroupsInvitationsEndpoint : ", err.Error())
				return getGroupsInvitationsResponse{invitations, err.Error()}, nil
			}
			invitations[idx].From = tmpInvitation
		}

		return getGroupsInvitationsResponse{invitations, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetGroupsInvitationsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getGroupsInvitationsRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPGetGroupsInvitationsRequest 1 : ", err.Error())
		return nil, err
	}

	groupID, err := strconv.ParseInt(mux.Vars(r)["UserID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPGetGroupsInvitationsRequest 2 : ", err.Error())
		return nil, err
	}
	(&request).UserID = groupID

	return request, nil
}

func DecodeHTTPGetGroupsInvitationsResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getGroupsInvitationsResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPGetGroupsInvitationsResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func GetGroupsInvitationsHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").
		Path("/groups/{UserID}/invitations/groups").
		Handler(httptransport.NewServer(
			endpoints.GetGroupsInvitationsEndpoint,
			DecodeHTTPGetGroupsInvitationsRequest,
			EncodeHTTPGenericResponse,
			append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetGroupsInvitations", logger)))...,
		))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetGroupsInvitations(ctx context.Context, groupID int64) ([]GroupInvitation, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getGroupsInvitations",
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetGroupsInvitations(ctx, groupID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetGroupsInvitations(ctx context.Context, groupID int64) ([]GroupInvitation, error) {
	v, err := mw.next.GetGroupsInvitations(ctx, groupID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetGroupsInvitationsEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetGroupsInvitations")
		csLogger := log.With(logger, "method", "GetGroupsInvitations")

		csEndpoint = GetGroupsInvitationsEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetGroupsInvitations")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetGroupsInvitations(ctx context.Context, groupID int64) ([]GroupInvitation, error) {
	var et []GroupInvitation

	request := getGroupsInvitationsRequest{UserID: groupID}
	response, err := e.GetGroupsInvitationsEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error GetGroupsInvitations : ", err.Error())
		return et, err
	}
	et = response.(getGroupsInvitationsResponse).Invitations
	return et, str2err(response.(getGroupsInvitationsResponse).Err)
}

func EncodeHTTPGetGroupsInvitationsRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(getGroupsInvitationsRequest).UserID)
	encodedUrl, err := route.Path(r.URL.Path).URL("UserID", id)
	if err != nil {
		fmt.Println("Error EncodeHTTPGetGroupsInvitationsRequest : ", err.Error())
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetGroupsInvitations(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/groups/{UserID}/invitations/groups"),
		EncodeHTTPGetGroupsInvitationsRequest,
		DecodeHTTPGetGroupsInvitationsResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "GetGroupsInvitations")(ceEndpoint)
	return ceEndpoint, nil
}
