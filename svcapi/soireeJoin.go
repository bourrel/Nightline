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

	"svcdb"
)

/*************** Service ***************/
func (s Service) SoireeJoin(ctx context.Context, UserID, SoireeID int64) (svcdb.Soiree, string, error) {
	var soiree svcdb.Soiree
	var token string

	//TODO: vérifier qu'il y ai autant de leave que de join (sinon leave d'abord)
	soiree, _, err := s.svcdb.UserJoinSoiree(ctx, UserID, SoireeID)
	if err != nil {
		fmt.Println("Error SoireeJoin " + err.Error())
		return soiree, "", dbToHTTPErr(err)
	}

	token = "0a0zda645d0az4d0a640er64g0e"
	return soiree, token, nil
}

/*************** Endpoint ***************/
type SoireeJoinRequest struct {
	SoireeID int64 `json:"soireeID"`
	UserID   int64 `json:"userID"`
}

type SoireeJoinResponse struct {
	Soiree svcdb.Soiree `json:"soiree"`
	Token  string       `json:"token"`
}

func SoireeJoinEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(SoireeJoinRequest)
		soiree, token, err := svc.SoireeJoin(ctx, req.UserID, req.SoireeID)
		return SoireeJoinResponse{Soiree: soiree, Token: token}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPSoireeJoinRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req SoireeJoinRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		fmt.Println("Error DecodeHTTPSoireeJoinRequest 1" + err.Error())
		return req, RequestError
	}

	soireeID, err := strconv.ParseInt(mux.Vars(r)["SoireeId"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPSoireeJoinRequest 3" + err.Error())
		return req, RequestError
	}

	(&req).SoireeID = soireeID

	fmt.Println(req.SoireeID)
	fmt.Println(req.UserID)

	return req, nil
}

func DecodeHTTPSoireeJoinResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response SoireeJoinResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		panic(err)
		fmt.Println("Error DecodeHTTPSoireeJoinResponse " + err.Error())
		return nil, err
	}
	return response, nil
}

func SoireeJoinHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/soiree/{SoireeId:[0-9]+}/join").Handler(httptransport.NewServer(
		endpoints.SoireeJoinEndpoint,
		DecodeHTTPSoireeJoinRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "SoireeJoin", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) SoireeJoin(ctx context.Context, UserID, SoireeID int64) (svcdb.Soiree, string, error) {
	newSoiree, token, err := mw.next.SoireeJoin(ctx, UserID, SoireeID)

	mw.logger.Log(
		"method", "SoireeJoin",
		"request", SoireeJoinRequest{UserID: UserID, SoireeID: SoireeID},
		"response", newSoiree,
		"took", time.Since(time.Now()),
	)
	return newSoiree, token, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) SoireeJoin(ctx context.Context, UserID, SoireeID int64) (svcdb.Soiree, string, error) {
	return mw.next.SoireeJoin(ctx, UserID, SoireeID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) SoireeJoin(ctx context.Context, UserID, SoireeID int64) (svcdb.Soiree, string, error) {
	return mw.next.SoireeJoin(ctx, UserID, SoireeID)
}

/*************** Main ***************/
/* Main */
func BuildSoireeJoinEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "SoireeJoin")
		csLogger := log.With(logger, "method", "SoireeJoin")

		csEndpoint = SoireeJoinEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "SoireeJoin")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
