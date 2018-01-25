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
func (s Service) GetUserPreferences(ctx context.Context, userID int64) ([]svcdb.Preference, error) {
	preferences, err := s.svcdb.GetUserPreferences(ctx, userID)
	return preferences, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type GetUserPreferencesRequest struct {
	UserID int64 `json:"id"`
}

type GetUserPreferencesResponse struct {
	User []svcdb.Preference `json:"preference"`
}

func GetUserPreferencesEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetUserPreferencesRequest)
		pref, err := svc.GetUserPreferences(ctx, req.UserID)
		return GetUserPreferencesResponse{pref}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPGetUserPreferencesRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request GetUserPreferencesRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPGetUserPreferencesRequest 1 : ", err.Error())
		return nil, err
	}

	userID, err := strconv.ParseInt(mux.Vars(r)["UserId"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPGetUserPreferencesRequest 2 : ", err.Error())
		return nil, RequestError
	}
	(&request).UserID = userID

	return request, nil
}

func DecodeHTTPGetUserPreferencesResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response GetUserPreferencesResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPGetUserPreferencesResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func GetUserPreferencesHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/users/{UserId:[0-9]+}/preferences").Handler(httptransport.NewServer(
		endpoints.GetUserPreferencesEndpoint,
		DecodeHTTPGetUserPreferencesRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetUserPreferences", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetUserPreferences(ctx context.Context, userID int64) ([]svcdb.Preference, error) {
	preferences, err := mw.next.GetUserPreferences(ctx, userID)

	mw.logger.Log(
		"method", "GetUserPreferences",
		"userID", userID,
		"response", preferences,
		"took", time.Since(time.Now()),
	)
	return preferences, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) GetUserPreferences(ctx context.Context, userID int64) ([]svcdb.Preference, error) {
	return mw.next.GetUserPreferences(ctx, userID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetUserPreferences(ctx context.Context, userID int64) ([]svcdb.Preference, error) {
	return mw.next.GetUserPreferences(ctx, userID)
}

/*************** Main ***************/
/* Main */
func BuildGetUserPreferencesEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetUserPreferences")
		csLogger := log.With(logger, "method", "GetUserPreferences")

		csEndpoint = GetUserPreferencesEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetUserPreferences")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
