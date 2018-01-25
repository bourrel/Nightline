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
func (s Service) UpdateGroup(c context.Context, new Group) (Group, error) {
	var group Group

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("UpdateGroup (WaitConnection) : " + err.Error())
		return group, err
	}
	defer CloseConnection(conn)

	group, err = s.GetGroup(c, new.ID)
	if err != nil {
		fmt.Println("UpdateGroup (GetGroup) : " + err.Error())
		return group, err
	}

	group.UpdateFrom(new)

	stmt, err := conn.PrepareNeo(`
	MATCH (n:GROUP)
	WHERE ID(n) = {ID}
	SET
		n.Name = {Name},
		n.Description = {Description}
	RETURN n`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("UpdateGroup (PrepareNeo) : " + err.Error())
		return group, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"ID":          group.ID,
		"Name":        group.Name,
		"Description": group.Description,
	})

	if err != nil {
		fmt.Println("UpdateGroup (QueryNeo) : " + err.Error())
		return group, err
	}

	data, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("UpdateGroup (NextNeo) : " + err.Error())
		return group, err
	}

	(&group).NodeToGroup(data[0].(graph.Node))
	return group, nil
}

/*************** Endpoint ***************/
type updateGroupRequest struct {
	Group Group `json:"group"`
}

type updateGroupResponse struct {
	Group Group  `json:"group"`
	Err   string `json:"err,omitempty"`
}

func UpdateGroupEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateGroupRequest)
		_, err := svc.UpdateGroup(ctx, req.Group)
		if err != nil {
			fmt.Println("UpdateGroupEndpoint: " + err.Error())
			return updateGroupResponse{Err: err.Error()}, nil
		}

		group, err := svc.GetGroup(ctx, req.Group.ID)
		if err != nil {
			fmt.Println("UpdateGroupEndpoint: " + err.Error())
			return updateGroupResponse{Group: group, Err: err.Error()}, nil
		}

		return updateGroupResponse{Group: group, Err: ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPUpdateGroupRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request updateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		fmt.Println("DecodeHTTPUpdateGroupRequest: " + err.Error())
		return nil, err
	}
	return request, nil
}

func DecodeHTTPUpdateGroupResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response updateGroupResponse

	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("DecodeHTTPUpdateGroupResponse: " + err.Error())
		return nil, err
	}

	return response, nil
}

func UpdateGroupHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/groups/update_group").Handler(httptransport.NewServer(
		endpoints.UpdateGroupEndpoint,
		DecodeHTTPUpdateGroupRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "UpdateGroup", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) UpdateGroup(ctx context.Context, u Group) (Group, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "updateGroup",
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.UpdateGroup(ctx, u)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) UpdateGroup(ctx context.Context, u Group) (Group, error) {
	v, err := mw.next.UpdateGroup(ctx, u)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildUpdateGroupEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "UpdateGroup")
		csLogger := log.With(logger, "method", "UpdateGroup")

		csEndpoint = UpdateGroupEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "UpdateGroup")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
func (e Endpoints) UpdateGroup(ctx context.Context, et Group) (Group, error) {
	request := updateGroupRequest{Group: et}
	response, err := e.UpdateGroupEndpoint(ctx, request)
	if err != nil {
		fmt.Println("UpdateGroup: " + err.Error())
		return et, err
	}
	return response.(updateGroupResponse).Group, str2err(response.(updateGroupResponse).Err)
}

func ClientUpdateGroup(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, "/groups/update_group"),
		EncodeHTTPGenericRequest,
		DecodeHTTPUpdateGroupResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "UpdateGroup")(ceEndpoint)
	return ceEndpoint, nil
}
