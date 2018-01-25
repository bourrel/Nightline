package svcapi

import (
	"context"
	"encoding/json"
	"net/http"
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
func (s Service) CreateNotification(ctx context.Context, name, text string, userID int64) (int32, int64, error) {
	return s.svcevent.Push(ctx, name, &messageBody{text}, userID)
}

/*************** Endpoint ***************/
type messageBody struct {
	Text string `json:"text"`
}

type createNotificationRequest struct {
	Name   string `json:"name"`
	Text   string `json:"text"`
	UserID int64  `json:"user"`
}

type createNotificationResponse struct {
	Partition int32 `json:"partiion"`
	Offset    int64 `json:"offset"`
}

func CreateNotificationEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createNotificationRequest)
		partition, offset, err := svc.CreateNotification(ctx, req.Name, req.Text, req.UserID)

		return createNotificationResponse{Partition: partition, Offset: offset}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPCreateNotificationRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req createNotificationRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return req, RequestError
	}
	return req, nil
}

func DecodeHTTPcreateNotificationResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response createNotificationResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func CreateNotificationHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/notification").Handler(httptransport.NewServer(
		endpoints.CreateNotificationEndpoint,
		DecodeHTTPCreateNotificationRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "CreateNotification", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) CreateNotification(ctx context.Context, name, text string, userID int64) (int32, int64, error) {
	partition, offset, err := mw.next.CreateNotification(ctx, name, text, userID)

	mw.logger.Log(
		"method", "CreateNotification",
		"name", name,
		"body", text,
		"took", time.Since(time.Now()),
	)
	return partition, offset, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) CreateNotification(ctx context.Context, name, text string, userID int64) (int32, int64, error) {
	return mw.next.CreateNotification(ctx, name, text, userID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) CreateNotification(ctx context.Context, name, text string, userID int64) (int32, int64, error) {
	return mw.next.CreateNotification(ctx, name, text, userID)
}

/*************** Main ***************/
/* Main */
func BuildCreateNotificationEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "CreateNotification")
		csLogger := log.With(logger, "method", "CreateNotification")

		csEndpoint = CreateNotificationEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "CreateNotification")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
