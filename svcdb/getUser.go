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
func (s Service) GetUser(_ context.Context, u User) (User, error) {
	var user User

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetUser (WaitConnection) : " + err.Error())
		return user, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
	MATCH (u:USER)
	WHERE u.Email = {email}
	RETURN u`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("GetUser (PrepareNeo) : " + err.Error())
		return user, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"email": u.Email,
	})
	if err != nil {
		fmt.Println("GetUser (QueryNeo) : " + err.Error())
		return user, err
	}

	row, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("GetUser (NextNeo) : " + err.Error())
		return user, err
	}

	(&user).NodeToUser(row[0].(graph.Node))

	if user.Password != u.Password {
		err = errors.New("Invalid password")
		return user, err
	}

	return user, nil
}

/*************** Endpoint ***************/
type getUserRequest struct {
	User User `json:"user"`
}

type getUserResponse struct {
	User User   `json:"user"`
	Err  string `json:"err,omitempty"`
}

func GetUserEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getUserRequest)
		user, err := svc.GetUser(ctx, req.User)
		if err != nil {
			return getUserResponse{user, err.Error()}, nil
		}
		return getUserResponse{user, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetUserRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getUserRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func DecodeHTTPGetUserResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getUserResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetUserHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/users/get_user").Handler(httptransport.NewServer(
		endpoints.GetUserEndpoint,
		DecodeHTTPGetUserRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetUser", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetUser(ctx context.Context, u User) (User, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getUser",
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetUser(ctx, u)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetUser(ctx context.Context, u User) (User, error) {
	v, err := mw.next.GetUser(ctx, u)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetUserEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetUser")
		csLogger := log.With(logger, "method", "GetUser")

		csEndpoint = GetUserEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetUser")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetUser(ctx context.Context, u User) (User, error) {
	request := getUserRequest{User: u}
	response, err := e.GetUserEndpoint(ctx, request)
	if err != nil {
		return u, err
	}
	return response.(getUserResponse).User, str2err(response.(getUserResponse).Err)
}

func ClientGetUser(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, "/users/get_user"),
		EncodeHTTPGenericRequest,
		DecodeHTTPGetUserResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "GetUser")(ceEndpoint)
	return ceEndpoint, nil
}
