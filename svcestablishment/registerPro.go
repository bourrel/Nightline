package svcestablishment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"svcdb"
	"time"

	"github.com/go-kit/kit/auth/jwt"
	"github.com/go-kit/kit/tracing/opentracing"
	mux "github.com/gorilla/mux"
	stdopentracing "github.com/opentracing/opentracing-go"

	stdjwt "github.com/dgrijalva/jwt-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"
)

/*************** Service ***************/
/* Service - Business logic */
func (s Service) RegisterPro(ctx context.Context, old svcdb.Pro) (svcdb.Pro, string, error) {
	var pro svcdb.Pro

	err := checkMandatoryProParams(old)
	if err != nil {
		fmt.Println("Error RegisterPro (checkMandatoryProParams)")
		return pro, "", err
	}

	pro, err = s.svcpayment.RegisterPro(ctx, old)
	if err != nil {
		fmt.Println("Error RegisterPro (SvcPaymentRegisterPro) : " + err.Error())
		return pro, "", err
	}
	
	pro, err = s.svcdb.CreatePro(ctx, pro)
	if err != nil {
		fmt.Println("Error RegisterPro (CreatePro)")
		return pro, "", dbToHTTPErr(err)
	}

	// Create a jwt token
	signer := stdjwt.NewWithClaims(stdjwt.SigningMethodHS256, claims)
	token, err := signer.SignedString(signKey)
	if err != nil {
		fmt.Println("Error RegisterPro (SignedString)")
		return pro, "", err
	}

	return pro, token, err
}

// Merged : Service definition

/*************** Endpoint ***************/
/* Endpoint - Req/Resp */
type RegisterProRequest struct {
	Pro svcdb.Pro `json:"pro"`
}
type RegisterProResponse struct {
	Pro   svcdb.Pro
	Token string
}

/* Endpoint - Create endpoint */
func RegisterProEndpoint(s IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		csReq := request.(RegisterProRequest)
		pro, token, err := s.RegisterPro(ctx, csReq.Pro)
		return RegisterProResponse{
			Pro:   pro,
			Token: token,
		}, err
	}
}

// Merged : endpoints struct

/*************** Transport ***************/
/* Transport - *coder Request */
func DecodeHTTPRegisterProRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req RegisterProRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

/* Transport - *coder Response */
func DecodeHTTPRegisterProResponse(_ context.Context, r *http.Response) (interface{}, error) {
	if r.StatusCode != http.StatusOK {
		return nil, errorDecoder(r)
	}
	var resp RegisterProResponse
	err := json.NewDecoder(r.Body).Decode(&resp)
	if err != nil {
		fmt.Println("Error DecodeHTTPRegisterProResponse")
		return nil, RequestError
	}
	return resp, err
}

func RegisterProHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/registerPro").Handler(httptransport.NewServer(
		endpoints.RegisterProEndpoint,
		DecodeHTTPRegisterProRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "RegisterPro", logger), jwt.HTTPToContext()))...,
	))
	return route
}

// Merged : HTTPHandler, errorWrapper struct */

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) RegisterPro(ctx context.Context, old svcdb.Pro) (svcdb.Pro, string, error) {
	new, token, err := mw.next.RegisterPro(ctx, old)

	mw.logger.Log(
		"method", "RegisterPro",
		"request", old,
		"response", RegisterProResponse{Pro: new, Token: token},
		"took", time.Since(time.Now()),
	)
	return new, token, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) RegisterPro(ctx context.Context, old svcdb.Pro) (svcdb.Pro, string, error) {
	return mw.next.RegisterPro(ctx, old)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) RegisterPro(ctx context.Context, old svcdb.Pro) (svcdb.Pro, string, error) {
	return mw.next.RegisterPro(ctx, old)
}

/*************** Main ***************/
/* Main */
func BuildRegisterProEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "RegisterPro")
		csLogger := log.With(logger, "method", "RegisterPro")

		csEndpoint = RegisterProEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "RegisterPro")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
