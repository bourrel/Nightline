package svcapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"svcws"
	"time"

	"github.com/go-kit/kit/auth/jwt"
	"github.com/go-kit/kit/tracing/opentracing"
	mux "github.com/gorilla/mux"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"

	"svcdb"
)

func firstGroupCreatedSuccess(s Service, ctx context.Context, ownerID int64) {
	// Check if user has already this success
	userSuccess, err := s.svcdb.GetUserSuccess(ctx, ownerID)
	if err != nil {
		return
	}

	for _, success := range userSuccess {
		if success.Active == true && success.Value == "group_created" {
			return
		}
	}

	// Check if user has already this success
	newSuccess, err := s.svcdb.GetSuccessByValue(ctx, "group_created")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// Create and send success notification
	notif := &successNotif{
		Name:      newSuccess.Name,
		SuccessID: newSuccess.ID,
	}
	s.svcevent.Push(ctx, svcws.Success, notif, ownerID)

	err = s.svcdb.AddSuccess(ctx, ownerID, "group_created")
}

/*************** Service ***************/
func (s Service) CreateGroup(ctx context.Context, old svcdb.Group, ownerID int64) (svcdb.Group, error) {
	group, err := s.svcdb.CreateGroup(ctx, old, ownerID)
	if err != nil {
		return group, dbToHTTPErr(err)
	}

	firstGroupCreatedSuccess(s, ctx, ownerID)
	return group, nil
}

/*************** Endpoint ***************/
type createGroupRequest struct {
	Group   svcdb.Group `json:"group"`
	OwnerID int64       `json:"ownerID"`
}

type createGroupResponse struct {
	Group svcdb.Group `json:"group"`
}

func CreateGroupEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createGroupRequest)
		group, err := svc.CreateGroup(ctx, req.Group, req.OwnerID)
		return createGroupResponse{Group: group}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPCreateGroupRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req createGroupRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return req, RequestError
	}
	return req, nil
}

func DecodeHTTPcreateGroupResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response createGroupResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func CreateGroupHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/groups").Handler(httptransport.NewServer(
		endpoints.CreateGroupEndpoint,
		DecodeHTTPCreateGroupRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "CreateGroup", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) CreateGroup(ctx context.Context, group svcdb.Group, ownerID int64) (svcdb.Group, error) {
	newGroup, err := mw.next.CreateGroup(ctx, group, ownerID)

	mw.logger.Log(
		"method", "CreateGroup",
		"request", group,
		"request", createGroupResponse{Group: newGroup},
		"took", time.Since(time.Now()),
	)
	return newGroup, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) CreateGroup(ctx context.Context, group svcdb.Group, ownerID int64) (svcdb.Group, error) {
	return mw.next.CreateGroup(ctx, group, ownerID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) CreateGroup(ctx context.Context, group svcdb.Group, ownerID int64) (svcdb.Group, error) {
	return mw.next.CreateGroup(ctx, group, ownerID)
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
