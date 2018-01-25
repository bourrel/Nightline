package svcestablishment

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
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
/* Service - Business logic */
func (s Service) GetStat(ctx context.Context, establishmentID, menuID int64, soireeBegin, soireeEnd time.Time) (int64, error) {
	// return s.svcsoiree.GetStat(ctx, establishmentID, menuID, soireeBegin, soireeEnd)
	return 0, dbToHTTPErr(nil)
}

// Merged : Service definition

/*************** Endpoint ***************/
/* Endpoint - Req/Resp */
type GetStatRequest struct {
	EstablishmentID int64
	MenuID          int64
	SoireeBegin     time.Time
	SoireeEnd       time.Time
}
type GetStatResponse struct {
	SoireeID int64
}

/* Endpoint - Create endpoint */
func GetStatEndpoint(s IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		csReq := request.(GetStatRequest)
		soireeID, err := s.GetStat(ctx, csReq.EstablishmentID, csReq.MenuID, csReq.SoireeBegin, csReq.SoireeEnd)
		return GetStatResponse{
			SoireeID: soireeID,
		}, err
	}
}

// Merged : endpoints struct

/*************** Transport ***************/
/* Transport - *coder Request */
func DecodeHTTPGetStatRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req GetStatRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, RequestError
	}
	return req, err
}

/* Transport - *coder Response */
func DecodeHTTPGetStatResponse(_ context.Context, r *http.Response) (interface{}, error) {
	if r.StatusCode != http.StatusOK {
		return nil, errorDecoder(r)
	}
	var resp GetStatResponse
	err := json.NewDecoder(r.Body).Decode(&resp)
	return resp, err
}

func GetStatHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/stats").Handler(httptransport.NewServer(
		endpoints.GetStatEndpoint,
		DecodeHTTPGetStatRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetStat", logger), jwt.HTTPToContext()))...,
	))
	return route
}

// Merged : HTTPHandler, errorWrapper struct */

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetStat(ctx context.Context, establishmentID, menuID int64, soireeBegin, soireeEnd time.Time) (soireeID int64, err error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "GetStat",
			"establishmentID", establishmentID, "menuID", menuID,
			"soireeBegin", soireeBegin, "soireeEnd", soireeEnd,
			"soireeID", soireeID,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetStat(ctx, establishmentID, menuID, soireeBegin, soireeEnd)
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) GetStat(ctx context.Context, establishmentID, menuID int64, soireeBegin, soireeEnd time.Time) (soireeID int64, err error) {
	return mw.next.GetStat(ctx, establishmentID, menuID, soireeBegin, soireeEnd)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetStat(ctx context.Context, establishmentID, menuID int64, soireeBegin, soireeEnd time.Time) (int64, error) {
	return mw.next.GetStat(ctx, establishmentID, menuID, soireeBegin, soireeEnd)
}

/*************** Main ***************/
/* Main */
func BuildGetStatEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetStat")
		csLogger := log.With(logger, "method", "GetStat")

		csEndpoint = GetStatEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetStat")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetStat(ctx context.Context, establishmentID, menuID int64, soireeBegin, soireeEnd time.Time) (int64, error) {
	var soireeID int64

	request := GetStatRequest{
		EstablishmentID: establishmentID,
		MenuID:          menuID,
		SoireeBegin:     soireeBegin,
		SoireeEnd:       soireeEnd,
	}
	response, err := e.GetStatEndpoint(ctx, request)
	if err != nil {
		return 0, err
	}
	soireeID = response.(GetStatResponse).SoireeID
	return soireeID, err
}

func ClientGetStat(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, "/GetStat"),
		EncodeHTTPGenericRequest,
		DecodeHTTPGetStatResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "GetStat")(gefmEndpoint)
	return gefmEndpoint, nil
}
