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
func (s Service) GetEstablishment(ctx context.Context, estabID int64) (svcdb.Establishment, error) {
	estab, err := s.svcdb.GetEstablishment(ctx, estabID)
	return estab, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type getEstablishmentRequest struct {
	EstabID int64 `json:"id"`
}

type getEstablishmentResponse struct {
	Establishment svcdb.Establishment `json:"establishment"`
}

func GetEstablishmentEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getEstablishmentRequest)
		establishment, err := svc.GetEstablishment(ctx, req.EstabID)
		return getEstablishmentResponse{establishment}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPGetEstablishmentRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getEstablishmentRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	estabID, err := strconv.ParseInt(mux.Vars(r)["EstabId"], 10, 64)
	if err != nil {
		fmt.Println(err)
		return nil, RequestError
	}
	(&request).EstabID = estabID

	return request, nil
}

func DecodeHTTPGetEstablishmentResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getEstablishmentResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetEstablishmentHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/establishments/{EstabId:[0-9]+}").Handler(httptransport.NewServer(
		endpoints.GetEstablishmentEndpoint,
		DecodeHTTPGetEstablishmentRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetEstablishment", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetEstablishment(ctx context.Context, estabID int64) (svcdb.Establishment, error) {
	estab, err := mw.next.GetEstablishment(ctx, estabID)

	mw.logger.Log(
		"method", "getEstablishment",
		"estabID", estabID,
		"response", estab,
		"took", time.Since(time.Now()),
	)

	return estab, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) GetEstablishment(ctx context.Context, estabID int64) (svcdb.Establishment, error) {
	return mw.next.GetEstablishment(ctx, estabID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetEstablishment(ctx context.Context, estabID int64) (svcdb.Establishment, error) {
	return mw.next.GetEstablishment(ctx, estabID)
}

/*************** Main ***************/
/* Main */
func BuildGetEstablishmentEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetEstablishment")
		csLogger := log.With(logger, "method", "GetEstablishment")

		csEndpoint = GetEstablishmentEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetEstablishment")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
