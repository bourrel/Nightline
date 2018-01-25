package svcestablishment

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
	"github.com/jasonwinn/geocoder"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"
)

/*************** Service ***************/
/* Service - Business logic */
func (s Service) UpdateEstab(ctx context.Context, estab svcdb.Establishment) (svcdb.Establishment, error) {
	if estab.Address != "" {
		lat, lng, err := geocoder.Geocode(estab.Address)

		estab.Lat = lat
		estab.Long = lng
		if err != nil {
			fmt.Println("Error CreateEstab 3 : ", err.Error())
		}
	}

	estab, err := s.svcdb.UpdateEstablishment(ctx, estab)
	return estab, dbToHTTPErr(err)
}

// Merged : Service definition

/*************** Endpoint ***************/
/* Endpoint - Req/Resp */
type UpdateEstabRequest struct {
	Establishment svcdb.Establishment
}
type UpdateEstabResponse struct {
	Establishment svcdb.Establishment
}

/* Endpoint - Create endpoint */
func UpdateEstabEndpoint(s IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		csReq := request.(UpdateEstabRequest)
		estab, err := s.UpdateEstab(ctx, csReq.Establishment)
		return UpdateEstabResponse{
			Establishment: estab,
		}, err
	}
}

// Merged : endpoints struct

/*************** Transport ***************/
/* Transport - *coder Request */
func DecodeHTTPUpdateEstabRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req UpdateEstabRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	estabID, err := strconv.ParseInt(mux.Vars(r)["EstabID"], 10, 64)
	if err != nil {
		return nil, RequestError
	}

	(&req).Establishment.ID = estabID

	return req, nil
}

/* Transport - *coder Response */
func DecodeHTTPUpdateEstabResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var resp UpdateEstabResponse

	if r.StatusCode != http.StatusOK {
		return nil, errorDecoder(r)
	}
	err := json.NewDecoder(r.Body).Decode(&resp)
	if err != nil {
		return nil, RequestError
	}
	return resp, err
}

func UpdateEstabHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/establishments/update/{EstabID}").Handler(httptransport.NewServer(
		endpoints.UpdateEstabEndpoint,
		DecodeHTTPUpdateEstabRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "UpdateEstab", logger), jwt.HTTPToContext()))...,
	))
	return route
}

// Merged : HTTPHandler, errorWrapper struct */

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) UpdateEstab(ctx context.Context, oldEstab svcdb.Establishment) (svcdb.Establishment, error) {
	newEstab, err := mw.next.UpdateEstab(ctx, oldEstab)

	mw.logger.Log(
		"method", "UpdateEstab",
		"request", oldEstab,
		"response", newEstab,
		"took", time.Since(time.Now()),
	)
	return newEstab, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) UpdateEstab(ctx context.Context, establishment svcdb.Establishment) (svcdb.Establishment, error) {
	return mw.next.UpdateEstab(ctx, establishment)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) UpdateEstab(ctx context.Context, establishment svcdb.Establishment) (svcdb.Establishment, error) {
	return mw.next.UpdateEstab(ctx, establishment)
}

/*************** Main ***************/
/* Main */
func BuildUpdateEstabEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "UpdateEstab")
		csLogger := log.With(logger, "method", "UpdateEstab")

		csEndpoint = UpdateEstabEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "UpdateEstab")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
