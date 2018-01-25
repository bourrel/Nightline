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
func (s Service) UpdateGroup(ctx context.Context, group svcdb.Group) (svcdb.Group, error) {
	group, err := s.svcdb.UpdateGroup(ctx, group)
	return group, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type UpdateGroupRequest struct {
	Group   svcdb.Group `json:"group"`
	GroupID int64       `json:"id"`
}

type UpdateGroupResponse struct {
	Group svcdb.Group `json:"group"`
}

func UpdateGroupEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(UpdateGroupRequest)

		group, err := svc.UpdateGroup(ctx, req.Group)
		if err != nil {
			fmt.Println("Error UpdateGroupEndpoint : ", err.Error())
			return UpdateGroupResponse{Group: group}, err
		}
		return UpdateGroupResponse{Group: group}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPUpdateGroupRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req UpdateGroupRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		fmt.Println("Error DecodeHTTPUpdateGroupRequest 1 : ", err.Error())
		return req, RequestError
	}

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPUpdateGroupRequest 2 : ", err.Error())
		return nil, err
	}

	groupID, err := strconv.ParseInt(mux.Vars(r)["GroupID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPUpdateGroupRequest 3 : ", err.Error())
		return nil, RequestError
	}
	(&req).Group.ID = groupID

	return req, nil
}

func DecodeHTTPUpdateGroupResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response UpdateGroupResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPUpdateGroupResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func UpdateGroupHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("PATCH").Path("/groups/{GroupID:[0-9]+}").Handler(httptransport.NewServer(
		endpoints.UpdateGroupEndpoint,
		DecodeHTTPUpdateGroupRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "UpdateGroup", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) UpdateGroup(ctx context.Context, Groups svcdb.Group) (svcdb.Group, error) {
	newPref, err := mw.next.UpdateGroup(ctx, Groups)

	mw.logger.Log(
		"method", "UpdateGroup",
		"request", Groups,
		"response", newPref,
		"took", time.Since(time.Now()),
	)
	return newPref, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) UpdateGroup(ctx context.Context, Groups svcdb.Group) (svcdb.Group, error) {
	return mw.next.UpdateGroup(ctx, Groups)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) UpdateGroup(ctx context.Context, Groups svcdb.Group) (svcdb.Group, error) {
	return mw.next.UpdateGroup(ctx, Groups)
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
