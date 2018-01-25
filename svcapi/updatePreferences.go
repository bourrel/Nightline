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
func (s Service) UpdatePreferences(ctx context.Context, userID int64, preferences []string) ([]string, error) {
	preferences, err := s.svcdb.UpdatePreference(ctx, userID, preferences)
	return preferences, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type UpdatePreferencesRequest struct {
	Preferences []string `json:"preferences"`
	UserID      int64    `json:"userID"`
}

type UpdatePreferencesResponse struct {
	Preferences []string `json:"preferences"`
}

func UpdatePreferencesEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		// var prefs []string

		req := request.(UpdatePreferencesRequest)
		// for i := 0; i < len(req.Preferences); i++ {
		// 	prefs = append(prefs, req.Preferences[i].Name)
		// }

		preferences, err := svc.UpdatePreferences(ctx, req.UserID, req.Preferences)
		if err != nil {
			fmt.Println("Error UpdatePreferencesEndpoint : ", err.Error())
			return UpdatePreferencesResponse{Preferences: preferences}, err
		}
		return UpdatePreferencesResponse{Preferences: preferences}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPUpdatePreferencesRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req UpdatePreferencesRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		fmt.Println("Error DecodeHTTPUpdatePreferencesRequest 1 : ", err.Error())
		return req, RequestError
	}

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPUpdatePreferencesRequest 2 : ", err.Error())
		return nil, err
	}

	userID, err := strconv.ParseInt(mux.Vars(r)["UserId"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPUpdatePreferencesRequest 3 : ", err.Error())
		return nil, RequestError
	}
	(&req).UserID = userID

	return req, nil
}

func DecodeHTTPUpdatePreferencesResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response UpdatePreferencesResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPUpdatePreferencesResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func UpdatePreferencesHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/users/{UserId:[0-9]+}/preferences").Handler(httptransport.NewServer(
		endpoints.UpdatePreferencesEndpoint,
		DecodeHTTPUpdatePreferencesRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "UpdatePreferences", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) UpdatePreferences(ctx context.Context, userID int64, preferences []string) ([]string, error) {
	newPref, err := mw.next.UpdatePreferences(ctx, userID, preferences)

	mw.logger.Log(
		"method", "UpdatePreferences",
		"request", preferences,
		"response", newPref,
		"took", time.Since(time.Now()),
	)
	return newPref, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) UpdatePreferences(ctx context.Context, userID int64, preferences []string) ([]string, error) {
	return mw.next.UpdatePreferences(ctx, userID, preferences)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) UpdatePreferences(ctx context.Context, userID int64, preferences []string) ([]string, error) {
	return mw.next.UpdatePreferences(ctx, userID, preferences)
}

/*************** Main ***************/
/* Main */
func BuildUpdatePreferencesEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "UpdatePreferences")
		csLogger := log.With(logger, "method", "UpdatePreferences")

		csEndpoint = UpdatePreferencesEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "UpdatePreferences")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
