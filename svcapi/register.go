package svcapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	stdjwt "github.com/dgrijalva/jwt-go"

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
func (s Service) Register(ctx context.Context, old svcdb.User) (svcdb.User, string, error) {
	var user svcdb.User
	var token string

	err := checkMandatoryUserParams(old)
	if err != nil {
		return user, "", err
	}

	user, err = s.svcdb.CreateUser(ctx, old)
	if err != nil {
		return user, "", dbToHTTPErr(err)
	}

	// Create a jwt token
	signer := stdjwt.NewWithClaims(stdjwt.SigningMethodHS256, claims)
	token, err = signer.SignedString(signKey)
	if err != nil {
		return user, "", err
	}

	return user, token, nil
}

/*************** Endpoint ***************/
type RegisterRequest struct {
	User svcdb.User `json:"user"`
}

type RegisterResponse struct {
	User  svcdb.User `json:"user"`
	Token string     `json:"token"`
}

func RegisterEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(RegisterRequest)
		user, token, err := svc.Register(ctx, req.User)
		return RegisterResponse{User: user, Token: token}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPRegisterRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req RegisterRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return req, RequestError
	}
	return req, nil
}

func DecodeHTTPRegisterResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response RegisterResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func RegisterHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/register").Handler(httptransport.NewServer(
		endpoints.RegisterEndpoint,
		DecodeHTTPRegisterRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "Register", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) Register(ctx context.Context, user svcdb.User) (svcdb.User, string, error) {
	newUser, token, err := mw.next.Register(ctx, user)

	mw.logger.Log(
		"method", "Register",
		"request", user,
		"request", RegisterResponse{User: newUser, Token: token},
		"took", time.Since(time.Now()),
	)
	return newUser, token, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) Register(ctx context.Context, user svcdb.User) (svcdb.User, string, error) {
	return mw.next.Register(ctx, user)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) Register(ctx context.Context, user svcdb.User) (svcdb.User, string, error) {
	return mw.next.Register(ctx, user)
}

/*************** Main ***************/
/* Main */
func BuildRegisterEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "Register")
		csLogger := log.With(logger, "method", "Register")

		csEndpoint = RegisterEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "Register")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
