package svcdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

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
func (s Service) CreateGroup(c context.Context, g Group, userID int64) (Group, error) {
	var group Group

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("CreateGroup (WaitConnection) : " + err.Error())
		return group, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
	MATCH (u:USER) WHERE ID(u) = {userID}
	CREATE (g:GROUP {
		Name: {Name},
		Description: {Description}
	}),
	(u)-[:CREATE]->(g)
	RETURN g, u`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("CreateGroup (PrepareNeo) : " + err.Error())
		return group, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"userID":      userID,
		"Name":        g.Name,
		"Description": g.Description,
	})

	if err != nil {
		fmt.Println("CreateGroup (QueryNeo) : " + err.Error())
		return group, err
	}

	data, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("CreateGroup (NextNeo) : " + err.Error())
		return group, err
	}

	(&group).NodeToGroup(data[0].(graph.Node))
	(&group.Owner).NodeToProfile(data[1].(graph.Node))
	return group, err
}

/*************** Endpoint ***************/
type createGroupRequest struct {
	Group  Group `json:"group"`
	UserID int64 `json:"userID"`
}

type createGroupResponse struct {
	Group Group  `json:"group"`
	Err   string `json:"err,omitempty"`
}

func CreateGroupEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createGroupRequest)
		group, err := svc.CreateGroup(ctx, req.Group, req.UserID)
		if err != nil {
			return createGroupResponse{group, err.Error()}, nil
		}
		return createGroupResponse{group, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPCreateGroupRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request createGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func DecodeHTTPCreateGroupResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response createGroupResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func CreateGroupHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/groups/create_group").Handler(httptransport.NewServer(
		endpoints.CreateGroupEndpoint,
		DecodeHTTPCreateGroupRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "CreateGroup", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) CreateGroup(ctx context.Context, u Group, userID int64) (Group, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "createGroup",
			"group", u,
			"userID", userID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.CreateGroup(ctx, u, userID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) CreateGroup(ctx context.Context, u Group, userID int64) (Group, error) {
	v, err := mw.next.CreateGroup(ctx, u, userID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildCreateGroupEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "CreateGroup")
		csLogger := log.With(logger, "method", "CreateGroup")

		csEndpoint = CreateGroupEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "CreateGroup")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
func (e Endpoints) CreateGroup(ctx context.Context, et Group, userID int64) (Group, error) {
	request := createGroupRequest{Group: et, UserID: userID}
	response, err := e.CreateGroupEndpoint(ctx, request)
	if err != nil {
		return et, err
	}
	return response.(createGroupResponse).Group, str2err(response.(createGroupResponse).Err)
}

func ClientCreateGroup(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, "/groups/create_group"),
		EncodeHTTPGenericRequest,
		DecodeHTTPCreateGroupResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "CreateGroup")(ceEndpoint)
	return ceEndpoint, nil
}
