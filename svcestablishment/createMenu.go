package svcestablishment

import (
	"context"
	"encoding/json"
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
func (s Service) CreateMenu(ctx context.Context, establishmentID int64, menu svcdb.Menu) (svcdb.Menu, error) {
	m, err := s.svcdb.CreateMenu(ctx, establishmentID, menu)
	return m, dbToHTTPErr(err)
}

// Merged : Service definition

/*************** Endpoint ***************/
/* Endpoint - Req/Resp */
type CreateMenuRequest struct {
	EstablishmentID int64      `json:"establishmentID"`
	Menu            svcdb.Menu `json:"menu"`
}
type CreateMenuResponse struct {
	Menu svcdb.Menu
}

/* Endpoint - Create endpoint */
func CreateMenuEndpoint(s IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		csReq := request.(CreateMenuRequest)
		menu, err := s.CreateMenu(ctx, csReq.EstablishmentID, csReq.Menu)
		return CreateMenuResponse{
			Menu: menu,
		}, err
	}
}

// Merged : endpoints struct

/*************** Transport ***************/
/* Transport - *coder Request */
func DecodeHTTPCreateMenuRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req CreateMenuRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

/* Transport - *coder Response */
func DecodeHTTPCreateMenuResponse(_ context.Context, r *http.Response) (interface{}, error) {
	if r.StatusCode != http.StatusOK {
		return nil, errorDecoder(r)
	}
	var resp CreateMenuResponse
	err := json.NewDecoder(r.Body).Decode(&resp)
	if err != nil {
		return nil, RequestError
	}
	return resp, err
}

func CreateMenuHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/menus").Handler(httptransport.NewServer(
		endpoints.CreateMenuEndpoint,
		DecodeHTTPCreateMenuRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "CreateMenu", logger), jwt.HTTPToContext()))...,
	))
	return route
}

// Merged : HTTPHandler, errorWrapper struct */

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) CreateMenu(ctx context.Context, establishmentID int64, menu svcdb.Menu) (svcdb.Menu, error) {
	newMenu, err := mw.next.CreateMenu(ctx, establishmentID, menu)

	mw.logger.Log(
		"method", "CreateMenu",
		"request", menu,
		"response", newMenu,
		"took", time.Since(time.Now()),
	)
	return newMenu, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) CreateMenu(ctx context.Context, establishmentID int64, menu svcdb.Menu) (svcdb.Menu, error) {
	return mw.next.CreateMenu(ctx, establishmentID, menu)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) CreateMenu(ctx context.Context, establishmentID int64, menu svcdb.Menu) (svcdb.Menu, error) {
	return mw.next.CreateMenu(ctx, establishmentID, menu)
}

/*************** Main ***************/
/* Main */
func BuildCreateMenuEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "CreateMenu")
		csLogger := log.With(logger, "method", "CreateMenu")

		csEndpoint = CreateMenuEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "CreateMenu")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
