package svcdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"time"

	"github.com/go-kit/kit/tracing/opentracing"
	mux "github.com/gorilla/mux"
	"github.com/johnnadratowski/golang-neo4j-bolt-driver/structures/graph"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"
)

/*************** Service ***************/
func (s Service) CreateUser(c context.Context, u User) (User, error) {
	var user User

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("CreateUser (WaitConnection) : " + err.Error())
		return user, err
	}
	defer CloseConnection(conn)

	_, err = s.GetUser(c, u)
	if err == nil {
		err = errors.New("User already exists")
		return user, err
	}
	err = nil

	stmt, err := conn.PrepareNeo(`CREATE (u:USER {
		Email: {Email},
		Pseudo: {Pseudo},
		Password: {Password}
	}) RETURN u`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("CreateUser (PrepareNeo) : " + err.Error())
		return user, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"Email":  u.Email,
		"Pseudo": u.Pseudo,
		// "Password":   encryptPassword(u.Password),
		"Password": u.Password,
	})

	if err != nil {
		fmt.Println("CreateUser (QueryNeo) : " + err.Error())
		return user, err
	}

	data, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("CreateUser (NextNeo) : " + err.Error())
		return user, err
	}

	(&user).NodeToUser(data[0].(graph.Node))
	return user, err
}

/*************** Endpoint ***************/
type createUserRequest struct {
	User User `json:"user"`
}

type createUserResponse struct {
	User User   `json:"user"`
	Err  string `json:"err,omitempty"`
}

func CreateUserEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createUserRequest)
		user, err := svc.CreateUser(ctx, req.User)
		if err != nil {
			return createUserResponse{user, err.Error()}, nil
		}
		return createUserResponse{user, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPCreateUserRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func DecodeHTTPCreateUserResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response createUserResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func CreateUserHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/users/create_user").Handler(httptransport.NewServer(
		endpoints.CreateUserEndpoint,
		DecodeHTTPCreateUserRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "CreateUser", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) CreateUser(ctx context.Context, u User) (User, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "createUser",
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.CreateUser(ctx, u)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) CreateUser(ctx context.Context, u User) (User, error) {
	v, err := mw.next.CreateUser(ctx, u)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildCreateUserEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "CreateUser")
		csLogger := log.With(logger, "method", "CreateUser")

		csEndpoint = CreateUserEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "CreateUser")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
func (e Endpoints) CreateUser(ctx context.Context, et User) (User, error) {
	request := createUserRequest{User: et}
	response, err := e.CreateUserEndpoint(ctx, request)
	if err != nil {
		return et, err
	}
	return response.(createUserResponse).User, str2err(response.(createUserResponse).Err)
}

func ClientCreateUser(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, "/users/create_user"),
		EncodeHTTPGenericRequest,
		DecodeHTTPCreateUserResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "CreateUser")(ceEndpoint)
	return ceEndpoint, nil
}
