package svcapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"svcdb"
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
func (s Service) GetGroup(ctx context.Context, groupID int64) (svcdb.Group, error) {
	group, err := s.svcdb.GetGroup(ctx, groupID)
	return group, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type getGroupRequest struct {
	EstabID int64 `json:"id"`
}

type getGroupResponse struct {
	Group svcdb.Group `json:"group"`
}

func GetGroupEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getGroupRequest)
		group, err := svc.GetGroup(ctx, req.EstabID)
		return getGroupResponse{group}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPGetGroupRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getGroupRequest

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

func DecodeHTTPGetGroupResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getGroupResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetGroupHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/groups/{GroupID:[0-9]+}").Handler(httptransport.NewServer(
		endpoints.GetGroupEndpoint,
		DecodeHTTPGetGroupRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetGroup", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetGroup(ctx context.Context, groupID int64) (svcdb.Group, error) {
	group, err := mw.next.GetGroup(ctx, groupID)

	mw.logger.Log(
		"method", "getGroup",
		"groupID", groupID,
		"response", group,
		"took", time.Since(time.Now()),
	)

	return group, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) GetGroup(ctx context.Context, groupID int64) (svcdb.Group, error) {
	return mw.next.GetGroup(ctx, groupID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetGroup(ctx context.Context, groupID int64) (svcdb.Group, error) {
	return mw.next.GetGroup(ctx, groupID)
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
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
