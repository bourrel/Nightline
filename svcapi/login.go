package svcapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-kit/kit/tracing/opentracing"
	mux "github.com/gorilla/mux"
	stdopentracing "github.com/opentracing/opentracing-go"

	stdjwt "github.com/dgrijalva/jwt-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"

	"svcdb"
)

/*************** Service ***************/
func (s Service) Login(ctx context.Context, old svcdb.User) (svcdb.User, string, error) {
	var user svcdb.User
	var token string

	err := checkMandatoryUserParams(old)
	if err != nil {
		fmt.Println("Error Login 1 : ", err.Error())
		return user, "", err
	}

	user, err = s.svcdb.GetUser(ctx, old)
	if err != nil {
		fmt.Println("Error Login 2 : ", err.Error())
		return user, "", dbToHTTPErr(err)
	}

	// Create a jwt token
	signer := stdjwt.NewWithClaims(stdjwt.SigningMethodHS256, claims)
	token, err = signer.SignedString(signKey)
	if err != nil {
		fmt.Println("Error Login 3 : ", err.Error())
		return user, "", err
	}

	return user, token, nil
}

func checkMandatoryUserParams(u svcdb.User) error {
	if u.Email == "" || u.Password == "" {
		err := errors.New("Invalid model")
		return err
	}
	return nil
}

/*************** Endpoint ***************/
type LoginRequest struct {
	User svcdb.User `json:"user"`
}

type LoginResponse struct {
	User  svcdb.User `json:"user"`
	Token string     `json:"token"`
}

func LoginEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		var user svcdb.User
		user = request.(LoginRequest).User

		newUser, token, err := svc.Login(ctx, user)
		return LoginResponse{User: newUser, Token: token}, err
	}
}

/*************** Transport ***************/
func DecodeHTTPLoginRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req LoginRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		fmt.Println("Error DecodeHTTPLoginRequest : ", err.Error())
		return req, RequestError
	}
	return req, nil
}

func DecodeHTTPLoginResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response LoginResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPLoginResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func LoginHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/login").Handler(httptransport.NewServer(
		endpoints.LoginEndpoint,
		DecodeHTTPLoginRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "Login", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) Login(ctx context.Context, user svcdb.User) (svcdb.User, string, error) {
	u, token, err := mw.next.Login(ctx, user)

	mw.logger.Log(
		"method", "Login",
		"request", user,
		"response", LoginResponse{User: u, Token: token},
		"took", time.Since(time.Now()),
	)

	return u, token, err
}

/*************** Authentication ***************/
/* Authentication */
func (mw serviceAuthenticationMiddleware) Login(ctx context.Context, user svcdb.User) (svcdb.User, string, error) {
	return mw.next.Login(ctx, user)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) Login(ctx context.Context, user svcdb.User) (svcdb.User, string, error) {
	return mw.next.Login(ctx, user)
}

/*************** Main ***************/
/* Main */
func BuildLoginEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "Login")
		csLogger := log.With(logger, "method", "Login")

		csEndpoint = LoginEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "Login")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}
