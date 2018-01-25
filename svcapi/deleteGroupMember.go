package svcapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-kit/kit/auth/jwt"
	"github.com/go-kit/kit/tracing/opentracing"
	mux "github.com/gorilla/mux"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"
)

/*************** Service ***************/
func (s Service) DeleteGroupMember(ctx context.Context, groupID, userID int64) error {
	err := s.svcdb.DeleteGroupMember(ctx, groupID, userID)
	return dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type deleteGroupMemberRequest struct {
	GroupID int64 `json:"groupId"`
	UserID  int64 `json:"userId"`
}

type deleteGroupMemberResponse struct {
	Deleted bool `json:"deleted"`
}

func DeleteGroupMemberEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteGroupMemberRequest)
		err := svc.DeleteGroupMember(ctx, req.GroupID, req.UserID)
		if err != nil {
			return deleteGroupMemberResponse{false}, err
		}
		return deleteGroupMemberResponse{true}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPDeleteGroupMemberRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request deleteGroupMemberRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	groupID, err := strconv.ParseInt(mux.Vars(r)["GroupID"], 10, 64)
	if err != nil {
		fmt.Println(err)
		return nil, RequestError
	}
	(&request).GroupID = groupID

	userID, err := strconv.ParseInt(mux.Vars(r)["UserID"], 10, 64)
	if err != nil {
		fmt.Println(err)
		return nil, RequestError
	}
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
	route := r.Methods("DELETE").Path("/groups/{GroupID:[0-9]+}/member/{UserID:[0-9]+}").Handler(httptransport.NewServer(
		endpoints.DeleteGroupMemberEndpoint,
		DecodeHTTPDeleteGroupMemberRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "DeleteGroupMember", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) DeleteGroupMember(ctx context.Context, groupID, userID int64) error {
	err := mw.next.DeleteGroupMember(ctx, groupID, userID)

	mw.logger.Log(
		"method", "deleteGroupMember",
		"groupID", groupID,
		"took", time.Since(time.Now()),
	)

	return err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) DeleteGroupMember(ctx context.Context, groupID, userID int64) error {
	return mw.next.DeleteGroupMember(ctx, groupID, userID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) DeleteGroupMember(ctx context.Context, groupID, userID int64) error {
	return mw.next.DeleteGroupMember(ctx, groupID, userID)
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
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
