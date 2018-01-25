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
func (s Service) GetSoireeFromEstablishment(ctx context.Context, estabID int64) (svcdb.Soiree, error) {
	soiree, err := s.svcdb.GetEstablishmentSoiree(ctx, estabID)
	return soiree, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type getSoireeFromEstablishmentRequest struct {
	EstabID int64 `json:"id"`
}

type getSoireeFromEstablishmentResponse struct {
	Soiree svcdb.Soiree `json:"soiree"`
}

func GetSoireeFromEstablishmentEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getSoireeFromEstablishmentRequest)
		soiree, err := svc.GetSoireeFromEstablishment(ctx, req.EstabID)
		if err != nil {
			fmt.Println("Error GetSoireeFromEstablishmentEndpoint : ", err.Error())
			return getSoireeFromEstablishmentResponse{soiree}, err
		}
		return getSoireeFromEstablishmentResponse{soiree}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetSoireeFromEstablishmentRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getSoireeFromEstablishmentRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPGetSoireeFromEstablishmentRequest 1 : ", err.Error())
		return nil, err
	}

	estabID, err := strconv.ParseInt(mux.Vars(r)["EstabId"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPGetSoireeFromEstablishmentRequest 2 : ", err.Error())
		return nil, RequestError
	}
	(&request).EstabID = estabID

	return request, nil
}

func DecodeHTTPGetSoireeFromEstablishmentResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getSoireeFromEstablishmentResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPGetSoireeFromEstablishmentResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func GetSoireeFromEstablishmentHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").
		Path("/establishments/{EstabId:[0-9]+}/soiree").
		Handler(httptransport.NewServer(
			endpoints.GetSoireeFromEstablishmentEndpoint,
			DecodeHTTPGetSoireeFromEstablishmentRequest,
			EncodeHTTPGenericResponse,
			append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetSoireeFromEstablishment", logger), jwt.HTTPToContext()))...,
		))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetSoireeFromEstablishment(ctx context.Context, estabID int64) (svcdb.Soiree, error) {
	soiree, err := mw.next.GetSoireeFromEstablishment(ctx, estabID)

	mw.logger.Log(
		"method", "getSoireeFromEstablishment",
		"estabID", estabID,
		"response", soiree,
		"took", time.Since(time.Now()),
	)

	return soiree, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) GetSoireeFromEstablishment(ctx context.Context, estabID int64) (svcdb.Soiree, error) {
	return mw.next.GetSoireeFromEstablishment(ctx, estabID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetSoireeFromEstablishment(ctx context.Context, estabID int64) (svcdb.Soiree, error) {
	return mw.next.GetSoireeFromEstablishment(ctx, estabID)
}

/*************** Main ***************/
/* Main */
func BuildGetSoireeFromEstablishmentEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var gefmEndpoint endpoint.Endpoint
	{
		gefmDuration := duration.With("method", "GetSoireeFromEstablishment")
		gefmLogger := log.With(logger, "method", "GetSoireeFromEstablishment")

		gefmEndpoint = GetSoireeFromEstablishmentEndpoint(svc)
		gefmEndpoint = opentracing.TraceServer(tracer, "GetSoireeFromEstablishment")(gefmEndpoint)
		gefmEndpoint = EndpointLoggingMiddleware(gefmLogger)(gefmEndpoint)
		gefmEndpoint = EndpointAuthenticationMiddleware()(gefmEndpoint)
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}
