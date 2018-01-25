package svcdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
func (s Service) GetUserGroups(_ context.Context, userID int64) ([]GroupArrayElement, error) {
	var groups []GroupArrayElement
	var tmpGroups GroupArrayElement

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Fprintf(os.Stdout, "GetUserGroups (WaitConnection) : "+err.Error())
		return nil, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(` 
		MATCH (u:USER)-[:MEMBER|:CREATE]->(g:GROUP)
		MATCH (g)-[rel]-(:USER)
		WHERE ID(u) = {id} 
		RETURN g, count(rel) as rel_count
	`)
	defer stmt.Close()
	if err != nil {
		fmt.Fprintf(os.Stdout, "GetUserGroups (PrepareNeo) : "+err.Error())
		return groups, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": userID,
	})

	if err != nil {
		fmt.Fprintf(os.Stdout, "GetUserGroups (QueryNeo) : "+err.Error())
		return groups, err
	}

	row, _, err := rows.NextNeo()
	for row != nil && err == nil {

		if err != nil && err != io.EOF {
			fmt.Println("GetUserGroups (NextNeo)")
			panic(err)
		} else if err != io.EOF {
			(&tmpGroups).NodeToGroupArrayElement(row[0].(graph.Node))

			if userCount, ok := row[1].(int64); ok {
				tmpGroups.UserCount = userCount
			} else {
				tmpGroups.UserCount = 0
			}

			groups = append(groups, tmpGroups)
		}
		row, _, err = rows.NextNeo()
	}

	return groups, nil
}

/*************** Endpoint ***************/
type getUserGroupsRequest struct {
	UserID int64 `json:"id"`
}

type getUserGroupsResponse struct {
	Group []GroupArrayElement `json:"groups"`
	Err   string              `json:"err,omitempty"`
}

func GetUserGroupsEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getUserGroupsRequest)
		group, err := svc.GetUserGroups(ctx, req.UserID)
		if err != nil {
			fmt.Println("Error GetUserGroupsEndpoint : ", err.Error())
			return getUserGroupsResponse{group, err.Error()}, nil
		}

		return getUserGroupsResponse{group, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetUserGroupsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getUserGroupsRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPGetUserGroupsRequest 1 : ", err.Error())
		return nil, err
	}

	userID, err := strconv.ParseInt(mux.Vars(r)["UserID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPGetUserGroupsRequest 2 : ", err.Error())
		return nil, err
	}
	(&request).UserID = userID

	return request, nil
}

func DecodeHTTPGetUserGroupsResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getUserGroupsResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPGetUserGroupsResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func GetUserGroupsHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/users/{UserID:[0-9]+}/groups").Handler(httptransport.NewServer(
		endpoints.GetUserGroupsEndpoint,
		DecodeHTTPGetUserGroupsRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetUserGroups", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetUserGroups(ctx context.Context, userID int64) ([]GroupArrayElement, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getUserGroups",
			"userID", userID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetUserGroups(ctx, userID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetUserGroups(ctx context.Context, userID int64) ([]GroupArrayElement, error) {
	v, err := mw.next.GetUserGroups(ctx, userID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetUserGroupsEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var gefmEndpoint endpoint.Endpoint
	{
		gefmDuration := duration.With("method", "GetUserGroups")
		gefmLogger := log.With(logger, "method", "GetUserGroups")

		gefmEndpoint = GetUserGroupsEndpoint(svc)
		gefmEndpoint = opentracing.TraceServer(tracer, "GetUserGroups")(gefmEndpoint)
		gefmEndpoint = EndpointLoggingMiddleware(gefmLogger)(gefmEndpoint)
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetUserGroups(ctx context.Context, userID int64) ([]GroupArrayElement, error) {
	var s []GroupArrayElement

	request := getUserGroupsRequest{UserID: userID}
	response, err := e.GetUserGroupsEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error GetUserGroups : ", err.Error())
		return s, err
	}
	s = response.(getUserGroupsResponse).Group
	return s, str2err(response.(getUserGroupsResponse).Err)
}

func EncodeHTTPGetUserGroupsRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(getUserGroupsRequest).UserID)
	encodedUrl, err := route.Path(r.URL.Path).URL("UserID", id)
	if err != nil {
		fmt.Println("Error EncodeHTTPGetUserGroupsRequest : ", err.Error())
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetUserGroups(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/users/{UserID:[0-9]+}/groups"),
		EncodeHTTPGetUserGroupsRequest,
		DecodeHTTPGetUserGroupsResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "GetUserGroups")(gefmEndpoint)
	return gefmEndpoint, nil
}
