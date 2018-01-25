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
)

/*************** Service ***************/
func (s Service) DeleteGroup(_ context.Context, groupID int64) error {
	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("DeleteGroup (WaitConnection) : " + err.Error())
		return err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (g:GROUP)
		WHERE ID(g) = {id}
		DETACH DELETE g
		RETURN g
	`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("DeleteGroup (PrepareNeo) : " + err.Error())
		return err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": groupID,
	})

	if err != nil {
		fmt.Println("DeleteGroup (QueryNeo) : " + err.Error())
		return err
	}

	_, _, err = rows.NextNeo()
	if err != nil {
		fmt.Println("DeleteGroup (NextNeo) : " + err.Error())
		return err
	}

	return nil
}

/*************** Endpoint ***************/
type deleteGroupRequest struct {
	GroupID int64 `json:"id"`
}

type deleteGroupResponse struct {
	Err string `json:"err,omitempty"`
}

func DeleteGroupEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteGroupRequest)
		err := svc.DeleteGroup(ctx, req.GroupID)
		if err != nil {
			fmt.Println("Error DeleteGroupEndpoint 1 : ", err.Error())
			return deleteGroupResponse{err.Error()}, nil
		}

		return deleteGroupResponse{""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPDeleteGroupRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request deleteGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func DecodeHTTPDeleteGroupResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response deleteGroupResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func DeleteGroupHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("DELETE").Path("/groups/{id}").Handler(httptransport.NewServer(
		endpoints.DeleteGroupEndpoint,
		DecodeHTTPDeleteGroupRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "DeleteGroup", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) DeleteGroup(ctx context.Context, groupID int64) error {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "deleteGroup",
			"groupID", groupID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.DeleteGroup(ctx, groupID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) DeleteGroup(ctx context.Context, groupID int64) error {
	err := mw.next.DeleteGroup(ctx, groupID)
	mw.ints.Add(1)
	return err
}

/*************** Main ***************/
/* Main */
func BuildDeleteGroupEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "DeleteGroup")
		csLogger := log.With(logger, "method", "DeleteGroup")

		csEndpoint = DeleteGroupEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "DeleteGroup")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) DeleteGroup(ctx context.Context, etID int64) error {
	request := deleteGroupRequest{GroupID: etID}
	response, err := e.DeleteGroupEndpoint(ctx, request)
	if err != nil {
		return err
	}
	return str2err(response.(deleteGroupResponse).Err)
}

func ClientDeleteGroup(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"DELETE",
		copyURL(u, "/groups/{id}"),
		EncodeHTTPGenericRequest,
		DecodeHTTPDeleteGroupResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "DeleteGroup")(ceEndpoint)
	return ceEndpoint, nil
}
