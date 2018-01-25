package svcestablishment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
func (s Service) CreateConso(ctx context.Context, establishmentID, menuID int64, conso svcdb.Conso) (svcdb.Conso, error) {
	conso, err := s.svcdb.CreateConso(ctx, establishmentID, menuID, conso)
	return conso, dbToHTTPErr(err)
}

// Merged : Service definition

/*************** Endpoint ***************/
/* Endpoint - Req/Resp */
type CreateConsoRequest struct {
	EstablishmentID int64       `json:"establishmentID"`
	MenuID          int64       `json:"menuID"`
	Conso           svcdb.Conso `json:"conso"`
}
type CreateConsoResponse struct {
	Conso svcdb.Conso
}

/* Endpoint - Create endpoint */
func CreateConsoEndpoint(s IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		csReq := request.(CreateConsoRequest)
		conso, err := s.CreateConso(ctx, csReq.EstablishmentID, csReq.MenuID, csReq.Conso)

		if err != nil {
			fmt.Println("Error CreateConsoEndpoint : ", err.Error())
			return CreateConsoResponse{conso}, err
		}
		return CreateConsoResponse{conso}, nil
	}
}

// Merged : endpoints struct

/*************** Transport ***************/
/* Transport - *coder Request */
func DecodeHTTPCreateConsoRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req CreateConsoRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		fmt.Println("Error DecodeHTTPCreateConsoRequest : ", err.Error())
		return req, err
	}
	return req, err
}

/* Transport - *coder Response */
func DecodeHTTPCreateConsoResponse(_ context.Context, r *http.Response) (interface{}, error) {
	if r.StatusCode != http.StatusOK {
		return nil, errorDecoder(r)
	}
	var resp CreateConsoResponse
	err := json.NewDecoder(r.Body).Decode(&resp)
	if err != nil {
		fmt.Println("Error DecodeHTTPCreateConsoResponse : ", err.Error())
		return nil, RequestError
	}
	return resp, err
}

func CreateConsoHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/consos").Handler(httptransport.NewServer(
		endpoints.CreateConsoEndpoint,
		DecodeHTTPCreateConsoRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "CreateConso", logger), jwt.HTTPToContext()))...,
	))
	return route
}

// Merged : HTTPHandler, errorWrapper struct */

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) CreateConso(ctx context.Context, establishmentID, menuID int64, conso svcdb.Conso) (svcdb.Conso, error) {
	newConso, err := mw.next.CreateConso(ctx, establishmentID, menuID, conso)

	mw.logger.Log(
		"method", "CreateConso",
		"request", conso,
		"response", newConso,
		"took", time.Since(time.Now()),
	)
	return conso, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) CreateConso(ctx context.Context, establishmentID, menuID int64, conso svcdb.Conso) (svcdb.Conso, error) {
	return mw.next.CreateConso(ctx, establishmentID, menuID, conso)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) CreateConso(ctx context.Context, establishmentID, menuID int64, conso svcdb.Conso) (svcdb.Conso, error) {
	return mw.next.CreateConso(ctx, establishmentID, menuID, conso)
}

/*************** Main ***************/
/* Main */
func BuildCreateConsoEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "CreateConso")
		csLogger := log.With(logger, "method", "CreateConso")

		csEndpoint = CreateConsoEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "CreateConso")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
