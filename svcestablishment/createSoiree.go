package svcestablishment

import (
	"context"
	"encoding/json"
	"fmt"
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

	"svcdb"
)

/*************** Service ***************/
/* Service - Business logic */
func (s Service) CreateSoiree(ctx context.Context, establishmentID, menuID int64, soiree svcdb.Soiree) (int64, error) {
	soireeID, err := s.svcsoiree.CreateSoiree(ctx, establishmentID, menuID, soiree)
	return soireeID, dbToHTTPErr(err)
}

// Merged : Service definition

/*************** Endpoint ***************/
/* Endpoint - Req/Resp */
type createSoireeRequest struct {
	EstablishmentID int64        `json:"establishmentID"`
	MenuID          int64        `json:"menuID"`
	Soiree          svcdb.Soiree `json:"soiree"`
}

type createSoireeResponse struct {
	SoireeID int64
}

/* Endpoint - Create endpoint */
func CreateSoireeEndpoint(s IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		csReq := request.(createSoireeRequest)
		soireeID, err := s.CreateSoiree(ctx, csReq.EstablishmentID, csReq.MenuID, csReq.Soiree)
		if err != nil {
			fmt.Println("Error CreateSoireeEndpoint : ", err.Error())
			return createSoireeResponse{SoireeID: soireeID}, err
		}
		return createSoireeResponse{SoireeID: soireeID}, nil
	}
}

// Merged : endpoints struct

/*************** Transport ***************/
/* Transport - *coder Request */
func DecodeHTTPCreateSoireeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req createSoireeRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		fmt.Println("Error DecodeHTTPCreateSoireeRequest : ", err.Error())
		return req, err
	}
	return req, nil
}

/* Transport - *coder Response */
func DecodeHTTPCreateSoireeResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var resp createSoireeResponse
	if r.StatusCode != http.StatusOK {
		return nil, errorDecoder(r)
	}

	err := json.NewDecoder(r.Body).Decode(&resp)
	if err != nil {
		fmt.Println("Error DecodeHTTPCreateSoireeResponse : ", err.Error())
		return nil, RequestError
	}
	return resp, err
}

func CreateSoireeHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/soiree").Handler(httptransport.NewServer(
		endpoints.CreateSoireeEndpoint,
		DecodeHTTPCreateSoireeRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "CreateSoiree", logger), jwt.HTTPToContext()))...,
	))
	return route
}

// Merged : HTTPHandler, errorWrapper struct */

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) CreateSoiree(ctx context.Context, establishmentID, menuID int64, soiree svcdb.Soiree) (soireeID int64, err error) {
	newSoiree, err := mw.next.CreateSoiree(ctx, establishmentID, menuID, soiree)

	mw.logger.Log(
		"method", "createSoiree",
		"request", createSoireeRequest{EstablishmentID: establishmentID, MenuID: menuID, Soiree: soiree},
		"response", newSoiree,
		"took", time.Since(time.Now()),
	)
	return newSoiree, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) CreateSoiree(ctx context.Context, establishmentID, menuID int64, s svcdb.Soiree) (soireeID int64, err error) {
	return mw.next.CreateSoiree(ctx, establishmentID, menuID, s)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) CreateSoiree(ctx context.Context, establishmentID, menuID int64, soiree svcdb.Soiree) (int64, error) {
	v, err := mw.next.CreateSoiree(ctx, establishmentID, menuID, soiree)
	mw.createSoiree_all.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildCreateSoireeEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "CreateSoiree")
		csLogger := log.With(logger, "method", "CreateSoiree")

		csEndpoint = CreateSoireeEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "CreateSoiree")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
