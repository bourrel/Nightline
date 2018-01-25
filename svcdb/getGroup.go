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
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/johnnadratowski/golang-neo4j-bolt-driver/structures/graph"
)

/*************** Service ***************/
func (s Service) GetGroup(_ context.Context, groupID int64) (Group, error) {
	var group Group
	var tmpMember Profile

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetGroup (WaitConnection) : " + err.Error())
		return group, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (g:GROUP), (g)<-[:CREATE]-(o) WHERE ID(g) = {id}
		OPTIONAL MATCH (g)<-[:MEMBER]-(u)
		RETURN g, o, u
	`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("GetGroup (PrepareNeo) : " + err.Error())
		return group, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": groupID,
	})

	if err != nil {
		fmt.Println("GetGroup (QueryNeo) : " + err.Error())
		return group, err
	}

	row, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("GetGroup (NextNeo) : " + err.Error())
		return group, err
	}

	(&group).NodeToGroup(row[0].(graph.Node))
	// Owner
	if row[1] != nil {
		(&group.Owner).NodeToProfile(row[1].(graph.Node))
	}
	// Members

	for row != nil {
		if err != nil {
			fmt.Println("GetGroup (---) : " + err.Error())
			panic(err)
		} else if row[2] != nil {
			(&tmpMember).NodeToProfile(row[2].(graph.Node))
			group.Users = append(group.Users, tmpMember)
		}
		row, _, err = rows.NextNeo()
	}

	return group, nil
}

/*************** Endpoint ***************/
type getGroupRequest struct {
	EstabID int64 `json:"id"`
}

type getGroupResponse struct {
	Group Group  `json:"group"`
	Err   string `json:"err,omitempty"`
}

func GetGroupEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getGroupRequest)
		group, err := svc.GetGroup(ctx, req.EstabID)
		if err != nil {
			fmt.Println("Error GetGroupEndpoint 1 : ", err.Error())
			return getGroupResponse{group, err.Error()}, nil
		}

		return getGroupResponse{group, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetGroupRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func DecodeHTTPGetGroupResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getGroupResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetGroupHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/groups/{id}").Handler(httptransport.NewServer(
		endpoints.GetGroupEndpoint,
		DecodeHTTPGetGroupRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetGroup", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetGroup(ctx context.Context, groupID int64) (Group, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getGroup",
			"groupID", groupID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetGroup(ctx, groupID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetGroup(ctx context.Context, groupID int64) (Group, error) {
	v, err := mw.next.GetGroup(ctx, groupID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetGroupEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetGroup")
		csLogger := log.With(logger, "method", "GetGroup")

		csEndpoint = GetGroupEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetGroup")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetGroup(ctx context.Context, etID int64) (Group, error) {
	var et Group

	request := getGroupRequest{EstabID: etID}
	response, err := e.GetGroupEndpoint(ctx, request)
	if err != nil {
		return et, err
	}
	et = response.(getGroupResponse).Group
	return et, str2err(response.(getGroupResponse).Err)
}

func ClientGetGroup(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/groups/{id}"),
		EncodeHTTPGenericRequest,
		DecodeHTTPGetGroupResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "GetGroup")(ceEndpoint)
	return ceEndpoint, nil
}
