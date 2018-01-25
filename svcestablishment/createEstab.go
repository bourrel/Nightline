package svcestablishment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
func (s Service) CreateEstab(ctx context.Context, old svcdb.Establishment, proID int64) (svcdb.Establishment, error) {
	var estabs []svcdb.SearchResponse
	var newEstab svcdb.Establishment

	estabs, err := s.svcdb.SearchEstablishments(ctx, old.Name)
	if err != nil {
		fmt.Println("Error CreateEstab 1 : ", err.Error())
		return newEstab, dbToHTTPErr(err)
	}

	for i := 0; i < len(estabs); i++ {
		if estabs[i].Name == old.Name {
			err := errors.New("Establishment already existing")
			fmt.Println("Error CreateEstab 2 : ", err.Error())
			return newEstab, err
		}
	}

	if old.Address != "" {
		old.Lat, old.Long, err = geocoder.Geocode(old.Address)
		if err != nil {
			fmt.Println("Error CreateEstab 3 : ", err.Error())
		}
	}

	if old.Type == "" {
		old.Type = "Unknown"
	}

	newEstab, err = s.svcdb.CreateEstablishment(ctx, old, proID)
	newEstab.Type = old.Type
	return newEstab, dbToHTTPErr(err)
}

// Merged : Service definition

/*************** Endpoint ***************/
/* Endpoint - Req/Resp */
type CreateEstabRequest struct {
	Estab svcdb.Establishment `json:"establishment"`
	ProID int64               `json:"proID"`
}
type CreateEstabResponse struct {
	Establishment svcdb.Establishment
	Token         string
}

/* Endpoint - Create endpoint */
func CreateEstabEndpoint(s IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		csReq := request.(CreateEstabRequest)

		establishments, err := s.GetProEstablishments(ctx, csReq.ProID)
		if err != nil {
			return CreateEstabResponse{}, err
		}

		if len(establishments) > 0 {
			err = errors.New("You can create only one establishment per account")
			return CreateEstabResponse{}, err
		}

		estab, err := s.CreateEstab(ctx, csReq.Estab, csReq.ProID)
		if err != nil {
			return CreateEstabResponse{Establishment: estab}, err
		}
		return CreateEstabResponse{Establishment: estab}, err
	}
}

// Merged : endpoints struct

/*************** Transport ***************/
/* Transport - *coder Request */
func DecodeHTTPCreateEstabRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req CreateEstabRequest
	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		fmt.Println("Error DecodeHTTPCreateEstabRequest : ", err.Error())
		return req, err
	}

	return req, nil
}

/* Transport - *coder Response */
func DecodeHTTPCreateEstabResponse(_ context.Context, r *http.Response) (interface{}, error) {
	if r.StatusCode != http.StatusOK {
		return nil, errorDecoder(r)
	}
	var resp CreateEstabResponse
	err := json.NewDecoder(r.Body).Decode(&resp)
	if err != nil {
		return nil, RequestError
	}
	return resp, err
}

func CreateEstabHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/establishments").Handler(httptransport.NewServer(
		endpoints.CreateEstabEndpoint,
		DecodeHTTPCreateEstabRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "CreateEstab", logger), jwt.HTTPToContext()))...,
	))
	return route
}

// Merged : HTTPHandler, errorWrapper struct */

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) CreateEstab(ctx context.Context, estab svcdb.Establishment, proID int64) (svcdb.Establishment, error) {
	newEstab, err := mw.next.CreateEstab(ctx, estab, proID)

	mw.logger.Log(
		"method", "CreateEstab",
		"request", estab,
		"response", newEstab,
		"took", time.Since(time.Now()),
	)
	return estab, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) CreateEstab(ctx context.Context, estab svcdb.Establishment, proID int64) (svcdb.Establishment, error) {
	return mw.next.CreateEstab(ctx, estab, proID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) CreateEstab(ctx context.Context, estab svcdb.Establishment, proID int64) (svcdb.Establishment, error) {
	return mw.next.CreateEstab(ctx, estab, proID)
}

/*************** Main ***************/
/* Main */
func BuildCreateEstabEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "CreateEstab")
		csLogger := log.With(logger, "method", "CreateEstab")

		csEndpoint = CreateEstabEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "CreateEstab")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
