package svcapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"strconv"

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
func (s Service) GetRecommendation(ctx context.Context, userID int64) ([]svcdb.Establishment, error) {
	estabs, err := s.svcdb.GetRecommendation(ctx, userID)
	return estabs, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type getRecommendationRequest struct {
	UserID int64 `json:"id"`
}

type getRecommendationResponse struct {
	Establishments []svcdb.Establishment `json:"establishments"`
}

func GetRecommendationEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getRecommendationRequest)
		estabs, err := svc.GetRecommendation(ctx, req.UserID)
		if err != nil {
			fmt.Println("Error GetRecommendationEndpoint : ", err.Error())
			return nil, err
		}
		return getRecommendationResponse{Establishments: estabs}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPgetRecommendationRequest(_ context.Context, r *http.Request) (interface{}, error) {
    var request getRecommendationRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPGetRecommendationRequest 1 : ", err.Error())
		return nil, err
	}

	userID, err := strconv.ParseInt(mux.Vars(r)["UserId"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPGetRecommendationRequest 2 : ", err.Error())
		return nil, RequestError
	}
	(&request).UserID = userID

	return request, nil
	
}

func DecodeHTTPgetRecommendationResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getRecommendationResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPgetRecommendationResponse : ", err.Error())
		return nil, RequestError
	}
	return response, nil
}

func GetRecommendationHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/recommendation/{UserId:[0-9]+}").Handler(httptransport.NewServer(
		endpoints.GetRecommendationEndpoint,
		DecodeHTTPgetRecommendationRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetRecommendation", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetRecommendation(ctx context.Context, userID int64) ([]svcdb.Establishment, error) {
	estabs, err := mw.next.GetRecommendation(ctx, userID)

	mw.logger.Log(
		"method", "GetRecommendation",
		"response", estabs,
		"took", time.Since(time.Now()),
	)
	return estabs, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) GetRecommendation(ctx context.Context, userID int64) ([]svcdb.Establishment, error) {
	return mw.next.GetRecommendation(ctx, userID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetRecommendation(ctx context.Context, userID int64) ([]svcdb.Establishment, error) {
	return mw.next.GetRecommendation(ctx, userID)
}

/*************** Main ***************/
/* Main */
func BuildGetRecommendationEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetRecommendation")
		csLogger := log.With(logger, "method", "GetRecommendation")

		csEndpoint = GetRecommendationEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetRecommendation")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
