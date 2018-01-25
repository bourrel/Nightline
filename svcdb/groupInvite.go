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
	"github.com/johnnadratowski/golang-neo4j-bolt-driver/structures/graph"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"
)

/*************** Service ***************/
func (s Service) GroupInvite(_ context.Context, groupID, friendID int64) (string, int64, error) {
	var group Group

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Fprintf(os.Stdout, "GroupInvite (WaitConnection) : "+err.Error())
		return "", 0, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
	MATCH (g:GROUP) WHERE ID(g) = {groupID}
	MATCH (f:USER) WHERE ID(f) = {friendID}
	CREATE (g)-[i:INVITE {
		Date: {date}
	}]->(f) RETURN g, i`)
	defer stmt.Close()
	if err != nil {
		fmt.Println("Error GroupInvite (PrepareNeo) : " + err.Error())
		return "", 0, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"groupID":  groupID,
		"friendID": friendID,
		"date":     time.Now().Format(timeForm),
	})

	if err != nil {
		fmt.Println("Error GroupInvite (QueryNeo) : " + err.Error())
		return "", 0, err
	}

	data, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("Error GroupInvite (NextNeo) : " + err.Error())
		return "", 0, err
	}

	(&group).NodeToGroup(data[0].(graph.Node))
	invitID := data[1].(graph.Relationship).RelIdentity

	return group.Name, invitID, nil
}

/*************** Endpoint ***************/
type groupInviteRequest struct {
	GroupID  int64 `json:"groupID"`
	FriendID int64 `json:"friendID"`
}

type groupInviteResponse struct {
	Name         string `json:"name"`
	InvitationID int64  `json:"id"`
	Err          string `json:"err,omitempty"`
}

func GroupInviteEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupInviteRequest)
		name, invitationId, err := svc.GroupInvite(ctx, req.GroupID, req.FriendID)
		if err != nil {
			fmt.Println("Error GroupInviteEndpoint : ", err.Error())
			return groupInviteResponse{Err: err.Error()}, nil
		}

		return groupInviteResponse{Name: name, InvitationID: invitationId}, nil
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
		return nil, err
	}
	(&request).GroupID = groupID

	friendID, err := strconv.ParseInt(mux.Vars(r)["FriendID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPGroupInviteRequest 2 : ", err.Error())
		return nil, err
	}
	(&request).FriendID = friendID

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
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GroupInvite", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GroupInvite(ctx context.Context, groupID, friendID int64) (string, int64, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "groupInvite",
			"groupID", groupID,
			"friendID", friendID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GroupInvite(ctx, groupID, friendID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GroupInvite(ctx context.Context, groupID, friendID int64) (string, int64, error) {
	name, invitationID, err := mw.next.GroupInvite(ctx, groupID, friendID)
	mw.ints.Add(1)
	return name, invitationID, err
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
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GroupInvite(ctx context.Context, groupID, friendID int64) (string, int64, error) {
	request := groupInviteRequest{GroupID: groupID, FriendID: friendID}
	response, err := e.GroupInviteEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error GroupInvite : ", err.Error())
		return "", 0, err
	}

	resp := response.(groupInviteResponse)
	return resp.Name, resp.InvitationID, str2err(resp.Err)
}

func EncodeHTTPGroupInviteRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	uid := strconv.FormatInt(request.(groupInviteRequest).GroupID, 10)
	fid := strconv.FormatInt(request.(groupInviteRequest).FriendID, 10)

	encodedUrl, err := route.Path(r.URL.Path).URL("GroupID", uid, "FriendID", fid)
	if err != nil {
		fmt.Println("Error EncodeHTTPGroupInviteRequest : ", err.Error())
		return err
	}

	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGroupInvite(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/groups/{GroupID:[0-9]+}/invite/{FriendID:[0-9]+}"),
		EncodeHTTPGroupInviteRequest,
		DecodeHTTPGroupInviteResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "GroupInvite")(gefmEndpoint)
	return gefmEndpoint, nil
}
