package svcdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
func (s Service) DeleteGroupMember(_ context.Context, groupID, userID int64) error {
	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("DeleteGroupMember (WaitConnection) : " + err.Error())
		return err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (g:GROUP)-[m:MEMBER]-(u:USER)
		WHERE
			ID(g) = {groupID} AND
			ID(u) = {userID}
		DETACH DELETE m
		RETURN u, g
	`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("DeleteGroupMember (PrepareNeo) : " + err.Error())
		return err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"groupID": groupID,
		"userID":  userID,
	})

	if err != nil {
		fmt.Println("DeleteGroupMember (QueryNeo) : " + err.Error())
		return err
	}

	_, _, err = rows.NextNeo()
	if err != nil {
		fmt.Println("DeleteGroupMember (NextNeo) : " + err.Error())
		return err
	}

	return nil
}

/*************** Endpoint ***************/
type deleteGroupMemberRequest struct {
	GroupID int64 `json:"groupId"`
	UserID  int64 `json:"userId"`
}

type deleteGroupMemberResponse struct {
	Err string `json:"err,omitempty"`
}

func DeleteGroupMemberEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteGroupMemberRequest)
		err := svc.DeleteGroupMember(ctx, req.GroupID, req.UserID)
		if err != nil {
			fmt.Println("Error DeleteGroupMemberEndpoint 1 : ", err.Error())
			return deleteGroupMemberResponse{err.Error()}, nil
		}

		return deleteGroupMemberResponse{""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPDeleteGroupMemberRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request deleteGroupMemberRequest
	// if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
	// 	return nil, err
	// }

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	groupID, err := strconv.ParseInt(mux.Vars(r)["groupID"], 10, 64)
	if err != nil {
		return nil, err
	}
	userID, err := strconv.ParseInt(mux.Vars(r)["userID"], 10, 64)
	if err != nil {
		return nil, err
	}
	(&request).GroupID = groupID
	(&request).UserID = userID

	return request, nil
}

func DecodeHTTPDeleteGroupMemberResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response deleteGroupMemberResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func DeleteGroupMemberHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("DELETE").Path("/groups/{groupID}/member/{userID}").Handler(httptransport.NewServer(
		endpoints.DeleteGroupMemberEndpoint,
		DecodeHTTPDeleteGroupMemberRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "DeleteGroupMember", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) DeleteGroupMember(ctx context.Context, groupID, userID int64) error {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "deleteGroupMember",
			"groupID", groupID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.DeleteGroupMember(ctx, groupID, userID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) DeleteGroupMember(ctx context.Context, groupID, userID int64) error {
	err := mw.next.DeleteGroupMember(ctx, groupID, userID)
	mw.ints.Add(1)
	return err
}

/*************** Main ***************/
/* Main */
func BuildDeleteGroupMemberEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "DeleteGroupMember")
		csLogger := log.With(logger, "method", "DeleteGroupMember")

		csEndpoint = DeleteGroupMemberEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "DeleteGroupMember")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) DeleteGroupMember(ctx context.Context, groupID, userID int64) error {
	request := deleteGroupMemberRequest{groupID, userID}
	response, err := e.DeleteGroupMemberEndpoint(ctx, request)
	if err != nil {
		return err
	}
	return str2err(response.(deleteGroupMemberResponse).Err)
}

func EncodeHTTPDeleteGroupMemberRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	uid := fmt.Sprintf("%v", request.(deleteGroupMemberRequest).UserID)
	gid := fmt.Sprintf("%v", request.(deleteGroupMemberRequest).GroupID)
	encodedUrl, err := route.Path(r.URL.Path).URL("groupID", gid, "userID", uid)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientDeleteGroupMember(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"DELETE",
		copyURL(u, "/groups/{groupID}/member/{userID}"),
		EncodeHTTPDeleteGroupMemberRequest,
		DecodeHTTPDeleteGroupMemberResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "DeleteGroupMember")(ceEndpoint)
	return ceEndpoint, nil
}
