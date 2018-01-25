package svcapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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
func (s Service) RateEstablishment(ctx context.Context, estabID, userID, rate int64) (svcdb.Establishment, error) {
	estab, err := s.svcdb.RateEstablishment(ctx, estabID, userID, rate)
	return estab, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type rateEstablishmentRequest struct {
	EstabID int64 `json:"estabID"`
	UserID  int64 `json:"userID"`
	Rate    int64 `json:"rate"`
}

type rateEstablishmentResponse struct {
	Establishment svcdb.Establishment `json:"estab"`
}

func RateEstablishmentEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(rateEstablishmentRequest)
		estab, err := svc.RateEstablishment(ctx, req.EstabID, req.UserID, req.Rate)
		return rateEstablishmentResponse{Establishment: estab}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPRateEstablishmentRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request rateEstablishmentRequest

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		return request, RequestError
	}

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPGetUserFriendsRequest 1 : ", err.Error())
		return nil, err
	}

	(&request).UserID, err = strconv.ParseInt(mux.Vars(r)["UserID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPGetUserFriendsRequest 2 : ", err.Error())
		return nil, RequestError
	}
	(&request).EstabID, err = strconv.ParseInt(mux.Vars(r)["EstabID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPGetUserFriendsRequest 3 : ", err.Error())
		return nil, RequestError
	}

	return request, nil
}

func DecodeHTTPrateEstablishmentResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response rateEstablishmentResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func RateEstablishmentHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/users/{UserID:[0-9]+}/rate/{EstabID:[0-9]+}").Handler(httptransport.NewServer(
		endpoints.RateEstablishmentEndpoint,
		DecodeHTTPRateEstablishmentRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "RateEstablishment", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) RateEstablishment(ctx context.Context, estabID, userID, rate int64) (svcdb.Establishment, error) {
	newEstablishment, err := mw.next.RateEstablishment(ctx, estabID, userID, rate)

	mw.logger.Log(
		"method", "RateEstablishment",
		"request", rateEstablishmentRequest{EstabID: estabID, UserID: userID, Rate: rate},
		"response", rateEstablishmentResponse{Establishment: newEstablishment},
		"took", time.Since(time.Now()),
	)
	return newEstablishment, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) RateEstablishment(ctx context.Context, estabID, userID, rate int64) (svcdb.Establishment, error) {
	return mw.next.RateEstablishment(ctx, estabID, userID, rate)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) RateEstablishment(ctx context.Context, estabID, userID, rate int64) (svcdb.Establishment, error) {
	return mw.next.RateEstablishment(ctx, estabID, userID, rate)
}

/*************** Main ***************/
/* Main */
func BuildRateEstablishmentEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "RateEstablishment")
		csLogger := log.With(logger, "method", "RateEstablishment")

		csEndpoint = RateEstablishmentEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "RateEstablishment")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
