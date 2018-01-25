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
func (s Service) DeleteGroup(ctx context.Context, groupID int64) error {
	err := s.svcdb.DeleteGroup(ctx, groupID)
	return dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type deleteGroupRequest struct {
	EstabID int64 `json:"id"`
}

type deleteGroupResponse struct {
	Deleted bool `json:"deleted"`
}

func DeleteGroupEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteGroupRequest)
		err := svc.DeleteGroup(ctx, req.EstabID)
		if err != nil {
			return deleteGroupResponse{false}, err
		}
		return deleteGroupResponse{true}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPDeleteGroupRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request deleteGroupRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	groupID, err := strconv.ParseInt(mux.Vars(r)["GroupID"], 10, 64)
	if err != nil {
		fmt.Println(err)
		return nil, RequestError
	}
	(&request).EstabID = groupID

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
	route := r.Methods("DELETE").Path("/groups/{GroupID:[0-9]+}").Handler(httptransport.NewServer(
		endpoints.DeleteGroupEndpoint,
		DecodeHTTPDeleteGroupRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "DeleteGroup", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) DeleteGroup(ctx context.Context, groupID int64) error {
	err := mw.next.DeleteGroup(ctx, groupID)

	mw.logger.Log(
		"method", "deleteGroup",
		"groupID", groupID,
		"took", time.Since(time.Now()),
	)

	return err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) DeleteGroup(ctx context.Context, groupID int64) error {
	return mw.next.DeleteGroup(ctx, groupID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) DeleteGroup(ctx context.Context, groupID int64) error {
	return mw.next.DeleteGroup(ctx, groupID)
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
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
