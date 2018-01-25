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
func (s Service) GetMenuFromSoiree(ctx context.Context, soireeID int64) (svcdb.Menu, error) {
	menu, err := s.svcdb.GetMenuFromSoiree(ctx, soireeID)
	return menu, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type getMenuFromSoireeRequest struct {
	SoireeID int64 `json:"SoireeID"`
}

type getMenuFromSoireeResponse struct {
	Menu svcdb.Menu `json:"menu"`
}

func GetMenuFromSoireeEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getMenuFromSoireeRequest)
		menu, err := svc.GetMenuFromSoiree(ctx, req.SoireeID)
		return getMenuFromSoireeResponse{Menu: menu}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPGetMenuFromSoireeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getMenuFromSoireeRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPGetMenuFromSoireeRequest 1 : ", err.Error())
		return nil, err
	}

	soireeID, err := strconv.ParseInt(mux.Vars(r)["SoireeID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPGetMenuFromSoireeRequest 2 : ", err.Error())
		return nil, RequestError
	}
	(&request).SoireeID = soireeID

	return request, nil
}

func DecodeHTTPGetMenuFromSoireeResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getMenuFromSoireeResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPGetMenuFromSoireeResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func GetMenuFromSoireeHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/soiree/{SoireeID:[0-9]+}/menu").Handler(httptransport.NewServer(
		endpoints.GetMenuFromSoireeEndpoint,
		DecodeHTTPGetMenuFromSoireeRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetMenuFromSoiree", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetMenuFromSoiree(ctx context.Context, soireeID int64) (svcdb.Menu, error) {
	menu, err := mw.next.GetMenuFromSoiree(ctx, soireeID)

	mw.logger.Log(
		"method", "getMenuFromSoiree",
		"soireeID", soireeID,
		"response", menu,
		"took", time.Since(time.Now()),
	)
	return menu, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) GetMenuFromSoiree(ctx context.Context, soireeID int64) (svcdb.Menu, error) {
	return mw.next.GetMenuFromSoiree(ctx, soireeID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetMenuFromSoiree(ctx context.Context, soireeID int64) (svcdb.Menu, error) {
	return mw.next.GetMenuFromSoiree(ctx, soireeID)
}

/*************** Main ***************/
/* Main */
func BuildGetMenuFromSoireeEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var gefmEndpoint endpoint.Endpoint
	{
		gefmDuration := duration.With("method", "GetMenuFromSoiree")
		gefmLogger := log.With(logger, "method", "GetMenuFromSoiree")

		gefmEndpoint = GetMenuFromSoireeEndpoint(svc)
		gefmEndpoint = opentracing.TraceServer(tracer, "GetMenuFromSoiree")(gefmEndpoint)
		gefmEndpoint = EndpointLoggingMiddleware(gefmLogger)(gefmEndpoint)
		gefmEndpoint = EndpointAuthenticationMiddleware()(gefmEndpoint)
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}
