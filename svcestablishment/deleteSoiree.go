package svcestablishment

import (
	"context"
	"encoding/json"
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
func (s Service) DeleteSoiree(ctx context.Context, soireeID int64) error {
	err := s.svcdb.DeleteSoiree(ctx, soireeID)
	return dbToHTTPErr(err)
}

// Merged : Service definition

/*************** Endpoint ***************/
/* Endpoint - Req/Resp */
type DeleteSoireeRequest struct {
	SoireeID int64 `json:"soireeID"`
}
type DeleteSoireeResponse struct {
}

/* Endpoint - Create endpoint */
func DeleteSoireeEndpoint(s IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		csReq := request.(DeleteSoireeRequest)

		err := s.DeleteSoiree(ctx, csReq.SoireeID)
		if err != nil {
			return DeleteSoireeResponse{}, err
		}
		return DeleteSoireeResponse{}, err
	}
}

// Merged : endpoints struct

/*************** Transport ***************/
/* Transport - *coder Request */
func DecodeHTTPDeleteSoireeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req DeleteSoireeRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	soireeID, err := strconv.ParseInt(mux.Vars(r)["SoireeID"], 10, 64)
	if err != nil {
		return nil, RequestError
	}

	(&req).SoireeID = soireeID
	return req, nil
}

/* Transport - *coder Response */
func DecodeHTTPDeleteSoireeResponse(_ context.Context, r *http.Response) (interface{}, error) {
	if r.StatusCode != http.StatusOK {
		return nil, errorDecoder(r)
	}
	var resp DeleteSoireeResponse
	err := json.NewDecoder(r.Body).Decode(&resp)
	if err != nil {
		return nil, RequestError
	}
	return resp, err
}

func DeleteSoireeHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/soirees/delete/{SoireeID}").Handler(httptransport.NewServer(
		endpoints.DeleteSoireeEndpoint,
		DecodeHTTPDeleteSoireeRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "DeleteSoiree", logger), jwt.HTTPToContext()))...,
	))
	return route
}

// Merged : HTTPHandler, errorWrapper struct */

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) DeleteSoiree(ctx context.Context, soireeID int64) error {
	err := mw.next.DeleteSoiree(ctx, soireeID)

	mw.logger.Log(
		"method", "DeleteSoiree",
		"request", soireeID,
		"took", time.Since(time.Now()),
	)
	return err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) DeleteSoiree(ctx context.Context, soireeID int64) error {
	return mw.next.DeleteSoiree(ctx, soireeID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) DeleteSoiree(ctx context.Context, soireeID int64) error {
	return mw.next.DeleteSoiree(ctx, soireeID)
}

/*************** Main ***************/
/* Main */
func BuildDeleteSoireeEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "DeleteSoiree")
		csLogger := log.With(logger, "method", "DeleteSoiree")

		csEndpoint = DeleteSoireeEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "DeleteSoiree")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
