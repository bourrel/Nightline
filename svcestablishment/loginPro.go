package svcestablishment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"svcdb"
	"time"

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
func (s Service) LoginPro(ctx context.Context, old svcdb.Pro) (svcdb.Pro, string, error) {
	var pro svcdb.Pro

	err := checkMandatoryProParams(old)
	if err != nil {
		fmt.Println("Error LoginPro (checkMandatoryProParams)" + err.Error())
		return pro, "", err
	}

	pro, err = s.svcdb.GetPro(ctx, old)
	if err != nil {
		fmt.Println("Error LoginPro (GetPro)" + err.Error())
		return pro, "", dbToHTTPErr(err)
	}

	// Create a jwt token
	signer := stdjwt.NewWithClaims(stdjwt.SigningMethodHS256, claims)
	token, err := signer.SignedString(signKey)
	if err != nil {
		fmt.Println("Error LoginPro (SignedString)" + err.Error())
		return pro, "", err
	}

	return pro, token, nil
}

func checkMandatoryProParams(pro svcdb.Pro) error {
	if pro.Email == "" || pro.Password == "" {
		err := errors.New("Invalid model")
		return err
	}
	return nil
}

/*************** Endpoint ***************/
/* Endpoint - Req/Resp */
type LoginProRequest struct {
	Pro svcdb.Pro `json:"pro"`
}

type LoginProResponse struct {
	Pro   svcdb.Pro `json:"pro"`
	Token string    `json:"token"`
}

/* Endpoint - Create endpoint */
func LoginProEndpoint(s IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		csReq := request.(LoginProRequest)
		pro, token, err := s.LoginPro(ctx, csReq.Pro)
		if err != nil {
			return nil, err
		}

		return LoginProResponse{
			Pro:   pro,
			Token: token,
		}, err
	}
}

// Merged : endpoints struct

/*************** Transport ***************/
/* Transport - *coder Request */
func DecodeHTTPLoginProRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req LoginProRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

/* Transport - *coder Response */
func DecodeHTTPLoginProResponse(_ context.Context, r *http.Response) (interface{}, error) {
	if r.StatusCode != http.StatusOK {
		return nil, errorDecoder(r)
	}
	var resp LoginProResponse
	err := json.NewDecoder(r.Body).Decode(&resp)
	if err != nil {
		fmt.Println("Error DecodeHTTPLoginProResponse" + err.Error())
		return nil, RequestError
	}
	return resp, err
}

func LoginProHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/loginPro").Handler(httptransport.NewServer(
		endpoints.LoginProEndpoint,
		DecodeHTTPLoginProRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "LoginPro", logger)))...,
	))
	return route
}

// Merged : HTTPHandler, errorWrapper struct */

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) LoginPro(ctx context.Context, old svcdb.Pro) (svcdb.Pro, string, error) {
	pro, token, err := mw.next.LoginPro(ctx, old)

	mw.logger.Log(
		"method", "LoginPro",
		"request", old,
		"response", LoginProResponse{Pro: pro, Token: token},
		"took", time.Since(time.Now()),
	)
	return pro, token, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) LoginPro(ctx context.Context, old svcdb.Pro) (svcdb.Pro, string, error) {
	return mw.next.LoginPro(ctx, old)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) LoginPro(ctx context.Context, old svcdb.Pro) (svcdb.Pro, string, error) {
	return mw.next.LoginPro(ctx, old)
}

/*************** Main ***************/
/* Main */
func BuildLoginProEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "LoginPro")
		csLogger := log.With(logger, "method", "LoginPro")

		csEndpoint = LoginProEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "LoginPro")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
