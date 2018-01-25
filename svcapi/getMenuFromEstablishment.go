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
func (s Service) GetMenuFromEstablishment(ctx context.Context, estabID int64) ([]svcdb.Menu, error) {
	menu, err := s.svcdb.GetMenuFromEstablishment(ctx, estabID)
	return menu, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type getMenuFromEstablishmentRequest struct {
	EstabID int64 `json:"EstabID"`
}

type getMenuFromEstablishmentResponse struct {
	Menu []svcdb.Menu `json:"menu"`
}

func GetMenuFromEstablishmentEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getMenuFromEstablishmentRequest)
		menu, err := svc.GetMenuFromEstablishment(ctx, req.EstabID)
		return getMenuFromEstablishmentResponse{Menu: menu}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPGetMenuFromEstablishmentRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getMenuFromEstablishmentRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPGetMenuFromEstablishmentRequest 1 : ", err.Error())
		return nil, err
	}

	estabID, err := strconv.ParseInt(mux.Vars(r)["EstabId"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPGetMenuFromEstablishmentRequest 2 : ", err.Error())
		return nil, RequestError
	}
	(&request).EstabID = estabID

	return request, nil
}

func DecodeHTTPGetMenuFromEstablishmentResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getMenuFromEstablishmentResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPGetMenuFromEstablishmentResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func GetMenuFromEstablishmentHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/establishments/{EstabId:[0-9]+}/menu").Handler(httptransport.NewServer(
		endpoints.GetMenuFromEstablishmentEndpoint,
		DecodeHTTPGetMenuFromEstablishmentRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetMenuFromEstablishment", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetMenuFromEstablishment(ctx context.Context, estabID int64) ([]svcdb.Menu, error) {
	menu, err := mw.next.GetMenuFromEstablishment(ctx, estabID)

	mw.logger.Log(
		"method", "getMenuFromEstablishment",
		"estabID", estabID,
		"response", menu,
		"took", time.Since(time.Now()),
	)
	return menu, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) GetMenuFromEstablishment(ctx context.Context, estabID int64) ([]svcdb.Menu, error) {
	return mw.next.GetMenuFromEstablishment(ctx, estabID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetMenuFromEstablishment(ctx context.Context, estabID int64) ([]svcdb.Menu, error) {
	return mw.next.GetMenuFromEstablishment(ctx, estabID)
}

/*************** Main ***************/
/* Main */
func BuildGetMenuFromEstablishmentEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var gefmEndpoint endpoint.Endpoint
	{
		gefmDuration := duration.With("method", "GetMenuFromEstablishment")
		gefmLogger := log.With(logger, "method", "GetMenuFromEstablishment")

		gefmEndpoint = GetMenuFromEstablishmentEndpoint(svc)
		gefmEndpoint = opentracing.TraceServer(tracer, "GetMenuFromEstablishment")(gefmEndpoint)
		gefmEndpoint = EndpointLoggingMiddleware(gefmLogger)(gefmEndpoint)
		gefmEndpoint = EndpointAuthenticationMiddleware()(gefmEndpoint)
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}
