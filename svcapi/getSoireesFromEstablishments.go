package svcapi

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
func (s Service) GetSoireesFromEstablishments(ctx context.Context, estabID int64) ([]svcdb.Soiree, error) {
	soiree, err := s.svcdb.GetSoireesByEstablishment(ctx, estabID)
	return soiree, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type getSoireesFromEstablishmentsRequest struct {
	EstabID int64 `json:"id"`
}

type getSoireesFromEstablishmentsResponse struct {
	Soirees []svcdb.Soiree `json:"soirees"`
}

func GetSoireesFromEstablishmentsEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getSoireesFromEstablishmentsRequest)
		soirees, err := svc.GetSoireesFromEstablishments(ctx, req.EstabID)
		return getSoireesFromEstablishmentsResponse{soirees}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPGetSoireesFromEstablishmentsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getSoireesFromEstablishmentsRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	estabID, err := strconv.ParseInt(mux.Vars(r)["EstabID"], 10, 64)
	if err != nil {
		return request, RequestError
	}
	(&request).EstabID = estabID

	return request, nil
}

func DecodeHTTPGetSoireesFromEstablishmentsResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getSoireesFromEstablishmentsResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetSoireesFromEstablishmentsHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").
		Path("/establishments/soiree").
		Handler(httptransport.NewServer(
			endpoints.GetSoireesFromEstablishmentsEndpoint,
			DecodeHTTPGetSoireesFromEstablishmentsRequest,
			EncodeHTTPGenericResponse,
			append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetSoireesFromEstablishments", logger), jwt.HTTPToContext()))...,
		))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetSoireesFromEstablishments(ctx context.Context, estabID int64) ([]svcdb.Soiree, error) {
	soirees, err := mw.next.GetSoireesFromEstablishments(ctx, estabID)

	mw.logger.Log(
		"method", "getSoireesFromEstablishments",
		"estabID", estabID,
		"response", soirees,
		"took", time.Since(time.Now()),
	)
	return soirees, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) GetSoireesFromEstablishments(ctx context.Context, estabID int64) ([]svcdb.Soiree, error) {
	return mw.next.GetSoireesFromEstablishments(ctx, estabID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetSoireesFromEstablishments(ctx context.Context, estabID int64) ([]svcdb.Soiree, error) {
	return mw.next.GetSoireesFromEstablishments(ctx, estabID)
}

/*************** Main ***************/
/* Main */
func BuildGetSoireesFromEstablishmentsEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetSoireesFromEstablishments")
		csLogger := log.With(logger, "method", "GetSoireesFromEstablishments")

		csEndpoint = GetSoireesFromEstablishmentsEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetSoireesFromEstablishments")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
