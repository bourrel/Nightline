package svcestablishment

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
/* Service - Business logic */
func (s Service) DeleteEstab(ctx context.Context, estabID int64) error {
	err := s.svcdb.DeleteEstab(ctx, estabID)
	return dbToHTTPErr(err)
}

// Merged : Service definition

/*************** Endpoint ***************/
/* Endpoint - Req/Resp */
type DeleteEstabRequest struct {
	EstabID int64 `json:"estabID"`
}
type DeleteEstabResponse struct {
}

/* Endpoint - Create endpoint */
func DeleteEstabEndpoint(s IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		csReq := request.(DeleteEstabRequest)

		err := s.DeleteEstab(ctx, csReq.EstabID)
		if err != nil {
			return DeleteEstabResponse{}, err
		}
		return DeleteEstabResponse{}, err
	}
}

// Merged : endpoints struct

/*************** Transport ***************/
/* Transport - *coder Request */
func DecodeHTTPDeleteEstabRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request DeleteEstabRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	EstabID, err := strconv.ParseInt(mux.Vars(r)["EstabID"], 10, 64)
	if err != nil {
		fmt.Println(err)
		return nil, RequestError
	}
	(&request).EstabID = EstabID

	return request, nil
}

/* Transport - *coder Response */
func DecodeHTTPDeleteEstabResponse(_ context.Context, r *http.Response) (interface{}, error) {
	if r.StatusCode != http.StatusOK {
		return nil, errorDecoder(r)
	}
	var resp DeleteEstabResponse
	err := json.NewDecoder(r.Body).Decode(&resp)
	if err != nil {
		return nil, RequestError
	}
	return resp, err
}

func DeleteEstabHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/establishment/delete/{EstabID:[0-9]+}").Handler(httptransport.NewServer(
		endpoints.DeleteEstabEndpoint,
		DecodeHTTPDeleteEstabRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "DeleteEstab", logger), jwt.HTTPToContext()))...,
	))
	return route
}

// Merged : HTTPHandler, errorWrapper struct */

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) DeleteEstab(ctx context.Context, estabID int64) error {
	err := mw.next.DeleteEstab(ctx, estabID)

	mw.logger.Log(
		"method", "DeleteEstab",
		"request", estabID,
		"took", time.Since(time.Now()),
	)
	return err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) DeleteEstab(ctx context.Context, estabID int64) error {
	return mw.next.DeleteEstab(ctx, estabID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) DeleteEstab(ctx context.Context, estabID int64) error {
	return mw.next.DeleteEstab(ctx, estabID)
}

/*************** Main ***************/
/* Main */
func BuildDeleteEstabEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "DeleteEstab")
		csLogger := log.With(logger, "method", "DeleteEstab")

		csEndpoint = DeleteEstabEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "DeleteEstab")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
