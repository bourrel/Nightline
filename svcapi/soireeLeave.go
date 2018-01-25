package svcapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-kit/kit/tracing/opentracing"
	mux "github.com/gorilla/mux"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"
)

/*************** Service ***************/
func (s Service) SoireeLeave(ctx context.Context, UserID, SoireeID int64, token string) error {
	//TODO: vérifier qu'il y ai bien un lien entre l'user et la soiree
	// TODO: vérifier token

	_, _, err := s.svcdb.UserLeaveSoiree(ctx, UserID, SoireeID)
	if err != nil {
		return dbToHTTPErr(err)
	}

	return nil
}

/*************** Endpoint ***************/
type SoireeLeaveRequest struct {
	SoireeID int64  `json:"soireeID"`
	UserID   int64  `json:"userID"`
	Token    string `json:"token"`
}

type SoireeLeaveResponse struct {
}

func SoireeLeaveEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(SoireeLeaveRequest)
		err := svc.SoireeLeave(ctx, req.UserID, req.SoireeID, req.Token)
		return SoireeLeaveResponse{}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPSoireeLeaveRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req SoireeLeaveRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		fmt.Println("Error DecodeHTTPSoireeJoinRequest 1" + err.Error())
		return req, RequestError
	}

	soireeID, err := strconv.ParseInt(mux.Vars(r)["SoireeId"], 10, 64)
	if err != nil {
		fmt.Println(err)
		return nil, RequestError
	}
	(&req).SoireeID = soireeID

	fmt.Println("Soiree : ", req.SoireeID)
	fmt.Println("User : ", req.UserID)
	fmt.Println("Token : ", req.Token)

	return req, nil
}

func DecodeHTTPSoireeLeaveResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response SoireeLeaveResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func SoireeLeaveHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/soiree/{SoireeId:[0-9]+}/leave").Handler(httptransport.NewServer(
		endpoints.SoireeLeaveEndpoint,
		DecodeHTTPSoireeLeaveRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "SoireeLeave", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) SoireeLeave(ctx context.Context, UserID, SoireeID int64, token string) error {
	err := mw.next.SoireeLeave(ctx, UserID, SoireeID, token)

	mw.logger.Log(
		"method", "SoireeLeave",
		"request", SoireeLeaveRequest{UserID: UserID, SoireeID: SoireeID},
		"took", time.Since(time.Now()),
	)
	return err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) SoireeLeave(ctx context.Context, UserID, SoireeID int64, token string) error {
	return mw.next.SoireeLeave(ctx, UserID, SoireeID, token)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) SoireeLeave(ctx context.Context, UserID, SoireeID int64, token string) error {
	return mw.next.SoireeLeave(ctx, UserID, SoireeID, token)
}

/*************** Main ***************/
/* Main */
func BuildSoireeLeaveEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "SoireeLeave")
		csLogger := log.With(logger, "method", "SoireeLeave")

		csEndpoint = SoireeLeaveEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "SoireeLeave")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
