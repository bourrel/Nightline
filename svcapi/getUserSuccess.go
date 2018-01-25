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
func (s Service) GetUserSuccess(ctx context.Context, userID int64) ([]svcdb.Success, error) {
	success, err := s.svcdb.GetUserSuccess(ctx, userID)
	return success, dbToHTTPErr(err)
}

/*************** Endpoint ***************/
type GetUserSuccessRequest struct {
	UserID int64 `json:"id"`
}

type GetUserSuccessResponse struct {
	User []svcdb.Success `json:"success"`
}

func GetUserSuccessEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetUserSuccessRequest)
		user, err := svc.GetUserSuccess(ctx, req.UserID)
		if err != nil {
			fmt.Println("Error GetUserSuccessEndpoint : ", err.Error())
			return GetUserSuccessResponse{user}, err
		}
		return GetUserSuccessResponse{user}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetUserSuccessRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request GetUserSuccessRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPGetUserSuccessRequest 1 : ", err.Error())
		return nil, err
	}

	userID, err := strconv.ParseInt(mux.Vars(r)["UserId"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPGetUserSuccessRequest 2 : ", err.Error())
		return nil, RequestError
	}
	(&request).UserID = userID

	return request, nil
}

func DecodeHTTPGetUserSuccessResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response GetUserSuccessResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPGetUserSuccessResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func GetUserSuccessHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/users/{UserId:[0-9]+}/success").Handler(httptransport.NewServer(
		endpoints.GetUserSuccessEndpoint,
		DecodeHTTPGetUserSuccessRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetUserSuccess", logger), jwt.HTTPToContext()))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetUserSuccess(ctx context.Context, userID int64) ([]svcdb.Success, error) {
	success, err := mw.next.GetUserSuccess(ctx, userID)

	mw.logger.Log(
		"method", "GetUserSuccess",
		"userID", userID,
		"response", success,
		"took", time.Since(time.Now()),
	)
	return success, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) GetUserSuccess(ctx context.Context, userID int64) ([]svcdb.Success, error) {
	return mw.next.GetUserSuccess(ctx, userID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetUserSuccess(ctx context.Context, userID int64) ([]svcdb.Success, error) {
	return mw.next.GetUserSuccess(ctx, userID)
}

/*************** Main ***************/
/* Main */
func BuildGetUserSuccessEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetUserSuccess")
		csLogger := log.With(logger, "method", "GetUserSuccess")

		csEndpoint = GetUserSuccessEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetUserSuccess")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointAuthenticationMiddleware()(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
