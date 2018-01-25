package svcapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"svcws"
	"time"

	"github.com/go-kit/kit/tracing/opentracing"
	mux "github.com/gorilla/mux"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"

	"svcdb"
)

func profileCompleteSuccess(s Service, ctx context.Context, userID int64) {
	// Check if user has already this success
	userSuccess, err := s.svcdb.GetUserSuccess(ctx, userID)
	if err != nil {
		return
	}

	for _, success := range userSuccess {
		if success.Active == true && success.Value == "profile_complete" {
			return
		}
	}

	// Get user friends count
	userFriends, err := s.svcdb.GetUserFriends(ctx, userID)
	if err != nil || len(userFriends) < 1 {
		fmt.Println(err.Error())
		return
	}

	// Check if user has already this success
	newSuccess, err := s.svcdb.GetSuccessByValue(ctx, "profile_complete")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// Create and send success notification
	notif := &successNotif{
		Name:      newSuccess.Name,
		SuccessID: newSuccess.ID,
	}
	s.svcevent.Push(ctx, svcws.Success, notif, userID)

	err = s.svcdb.AddSuccess(ctx, userID, "profile_complete")
}

/*************** Service ***************/
func (s Service) UpdateUser(ctx context.Context, old svcdb.User) (svcdb.User, error) {
	var user svcdb.User

	fmt.Println("updateUser", old.Birthdate)

	user, err := s.svcdb.UpdateUser(ctx, old)
	if err != nil {
		fmt.Println("Error UpdateUser 2 : ", err.Error())
		return user, dbToHTTPErr(err)
	}

	if user.Complete() {
		profileCompleteSuccess(s, ctx, user.ID)
	}

	return user, nil
}

/*************** Endpoint ***************/
type UpdateUserRequest struct {
	User svcdb.User `json:"user"`
}

type UpdateUserResponse struct {
	User svcdb.User `json:"user"`
}

func UpdateUserEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(UpdateUserRequest)
		user, err := svc.UpdateUser(ctx, req.User)
		return UpdateUserResponse{User: user}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPUpdateUserRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req UpdateUserRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		fmt.Println("Error DecodeHTTPUpdateUserRequest 1 : ", err.Error())
		return req, RequestError
	}

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPUpdateUserRequest 2 : ", err.Error())
		return nil, err
	}

	userID, err := strconv.ParseInt(mux.Vars(r)["UserId"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPUpdateUserRequest 3 : ", err.Error())
		return nil, RequestError
	}
	(&req).User.ID = userID

	return req, nil
}

func DecodeHTTPUpdateUserResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response UpdateUserResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPUpdateUserResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func UpdateUserHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("PATCH").Path("/users/{UserId:[0-9]+}").Handler(httptransport.NewServer(
		endpoints.UpdateUserEndpoint,
		DecodeHTTPUpdateUserRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "UpdateUser", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) UpdateUser(ctx context.Context, user svcdb.User) (svcdb.User, error) {
	newUser, err := mw.next.UpdateUser(ctx, user)

	mw.logger.Log(
		"method", "UpdateUser",
		"request", user,
		"response", newUser,
		"took", time.Since(time.Now()),
	)
	return newUser, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) UpdateUser(ctx context.Context, user svcdb.User) (svcdb.User, error) {
	return mw.next.UpdateUser(ctx, user)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) UpdateUser(ctx context.Context, user svcdb.User) (svcdb.User, error) {
	return mw.next.UpdateUser(ctx, user)
}

/*************** Main ***************/
/* Main */
func BuildUpdateUserEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "UpdateUser")
		csLogger := log.With(logger, "method", "UpdateUser")

		csEndpoint = UpdateUserEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "UpdateUser")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
