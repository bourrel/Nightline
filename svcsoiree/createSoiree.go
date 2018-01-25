package svcsoiree

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

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
	if soiree.Begin.After(soiree.End) || soiree.Begin.Before(time.Now()) {
		return 0, SoireeDateErr
	}

	// check menu, establishment validity
	_, err := s.svcdb.GetEstablishmentFromMenu(ctx, menuID)
	if err != nil {
		return 0, InvalidSoireeErr
	}
	
	// check if establishment do another soiree at the same time
	collisions, err := s.svcdb.GetSoireesByEstablishment(ctx, establishmentID) // todo Improve by better call
	for _, collision := range collisions {
		if (soiree.Begin.After(collision.Begin) && soiree.Begin.Before(collision.End)) ||
			(soiree.End.After(collision.Begin) && soiree.End.Before(collision.End)) ||
			soiree.Begin.Before(collision.Begin) && soiree.End.After(collision.End) ||
			soiree.Begin.Equal(collision.Begin) || soiree.End.Equal(collision.End) {
			return 0, SoireeDateErr
		}
	}
	
	// do it
	soiree, err = s.svcdb.CreateSoiree(ctx, menuID, establishmentID, soiree)
	return soiree.ID, err
}

// Merged : Service definition

/*************** Endpoint ***************/
/* Endpoint - Req/Resp */
type createSoireeRequest struct {
	EstablishmentID int64 `json:"establishmentID"`
	MenuID int64 `json:"menuID"`
	Soiree svcdb.Soiree `json:"soiree"`
}
type createSoireeResponse struct {
	SoireeID int64
	Err      error
}

/* Endpoint - Create endpoint */
func CreateSoireeEndpoint(s IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		csReq := request.(createSoireeRequest)
		soireeID, err := s.CreateSoiree(ctx, csReq.EstablishmentID, csReq.MenuID, csReq.Soiree)
		return createSoireeResponse{
			SoireeID: soireeID,
			Err:      err,
		}, nil
	}
}

// Merged : endpoints struct

/*************** Transport ***************/
/* Transport - *coder Request */
func DecodeHTTPCreateSoireeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req createSoireeRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

/* Transport - *coder Response */
func DecodeHTTPCreateSoireeResponse(_ context.Context, r *http.Response) (interface{}, error) {
	if r.StatusCode != http.StatusOK {
		return nil, errorDecoder(r)
	}
	var resp createSoireeResponse
	err := json.NewDecoder(r.Body).Decode(&resp)
	return resp, err
}

func CreateSoireeHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/createSoiree").Handler(httptransport.NewServer(
		endpoints.CreateSoireeEndpoint,
		DecodeHTTPCreateSoireeRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "CreateSoiree", logger)))...,
	))
	return route
}

// Merged : HTTPHandler, errorWrapper struct */

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) CreateSoiree(ctx context.Context, establishmentID, menuID int64, soiree svcdb.Soiree) (soireeID int64, err error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "createSoiree",
			"establishmentID", establishmentID, "menuID", menuID,
			"soiree", soiree,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.CreateSoiree(ctx, establishmentID, menuID, soiree)
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
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) CreateSoiree(ctx context.Context, establishmentID, menuID int64, soiree svcdb.Soiree) (int64, error) {
	var soireeID int64

	request := createSoireeRequest{
		EstablishmentID: establishmentID,
		MenuID: menuID,
		Soiree: soiree,
	}
	response, err := e.CreateSoireeEndpoint(ctx, request)
	if err != nil {
		return 0, err
	}
	soireeID = response.(createSoireeResponse).SoireeID
	return soireeID, response.(createSoireeResponse).Err
}

func ClientCreateSoiree(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, "/createSoiree"),
		EncodeHTTPGenericRequest,
		DecodeHTTPCreateSoireeResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "CreateSoiree")(gefmEndpoint)
	return gefmEndpoint, nil
}
