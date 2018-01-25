package svcestablishment

import (
	"context"
	"encoding/json"
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
/* Service - Business logic */
func (s Service) UpdatePro(ctx context.Context, pro svcdb.Pro) (svcdb.Pro, error) {
	pro, err := s.svcdb.UpdatePro(ctx, pro)
	return pro, dbToHTTPErr(err)
}

// Merged : Service definition

/*************** Endpoint ***************/
/* Endpoint - Req/Resp */
type UpdateProRequest struct {
	Pro svcdb.Pro `json:"pro"`
}
type UpdateProResponse struct {
	Pro svcdb.Pro `json:"pro"`
}

/* Endpoint - Create endpoint */
func UpdateProEndpoint(s IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		csReq := request.(UpdateProRequest)
		pro, err := s.UpdatePro(ctx, csReq.Pro)
		return UpdateProResponse{
			Pro: pro,
		}, err
	}
}

// Merged : endpoints struct

/*************** Transport ***************/
/* Transport - *coder Request */
func DecodeHTTPUpdateProRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req UpdateProRequest
	err := json.NewDecoder(r.Body).Decode(&req)

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	proID, err := strconv.ParseInt(mux.Vars(r)["ProID"], 10, 64)
	if err != nil {
		return nil, RequestError
	}
	(&req).Pro.ID = proID

	return req, err
}

/* Transport - *coder Response */
func DecodeHTTPUpdateProResponse(_ context.Context, r *http.Response) (interface{}, error) {
	if r.StatusCode != http.StatusOK {
		return nil, errorDecoder(r)
	}
	var resp UpdateProResponse
	err := json.NewDecoder(r.Body).Decode(&resp)
	if err != nil {
		return nil, RequestError
	}
	return resp, err
}

func UpdateProHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/pros/update/{ProID}").Handler(httptransport.NewServer(
		endpoints.UpdateProEndpoint,
		DecodeHTTPUpdateProRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "UpdatePro", logger), jwt.HTTPToContext()))...,
	))
	return route
}

// Merged : HTTPHandler, errorWrapper struct */

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) UpdatePro(ctx context.Context, oldPro svcdb.Pro) (svcdb.Pro, error) {
	newPro, err := mw.next.UpdatePro(ctx, oldPro)

	mw.logger.Log(
		"method", "UpdatePro",
		"request", oldPro,
		"response", newPro,
		"took", time.Since(time.Now()),
	)
	return newPro, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) UpdatePro(ctx context.Context, pro svcdb.Pro) (svcdb.Pro, error) {
	return mw.next.UpdatePro(ctx, pro)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) UpdatePro(ctx context.Context, pro svcdb.Pro) (svcdb.Pro, error) {
	return mw.next.UpdatePro(ctx, pro)
}

/*************** Main ***************/
/* Main */
func BuildUpdateProEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "UpdatePro")
		csLogger := log.With(logger, "method", "UpdatePro")

		csEndpoint = UpdateProEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "UpdatePro")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
