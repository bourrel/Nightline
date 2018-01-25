package svcdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
func (s Service) UsersConnected(_ context.Context, userID, friendID int64) (bool, error) {

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("UsersConnected (WaitConnection) : " + err.Error())
		return false, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (u:USER)-[:KNOW|INVITE]-(f:USER)
		WHERE ID(u) = {userID} AND ID(f) = {friendID}
		RETURN u, f
	`)
	defer stmt.Close()
	if err != nil {
		fmt.Println("Error UsersConnected (PrepareNeo) : " + err.Error())
		return false, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"userID":   userID,
		"friendID": friendID,
		"date":     time.Now().Format(timeForm),
	})

	if err != nil {
		fmt.Println("Error UsersConnected (QueryNeo) : " + err.Error())
		return false, err
	}

	_, _, err = rows.NextNeo()
	if err != nil {
		fmt.Println("Error UsersConnected (NextNeo) : " + err.Error())
		return false, nil
	}

	return true, nil
}

/*************** Endpoint ***************/
type usersConnectedRequest struct {
	UserID   int64 `json:"userID"`
	FriendID int64 `json:"friendID"`
}

type usersConnectedResponse struct {
	Connected bool   `json:"connected,omitempty"`
	Err       string `json:"err,omitempty"`
}

func UsersConnectedEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(usersConnectedRequest)
		connected, err := svc.UsersConnected(ctx, req.UserID, req.FriendID)
		if err != nil {
			fmt.Println("Error UsersConnectedEndpoint : ", err.Error())
			return usersConnectedResponse{Connected: connected, Err: err.Error()}, nil
		}
		return usersConnectedResponse{Connected: connected}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPUsersConnectedRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request usersConnectedRequest

	if err := r.ParseForm(); err != nil {
		fmt.Println("Error DecodeHTTPUsersConnectedRequest 1 : ", err.Error())
		return nil, err
	}

	userID, err := strconv.ParseInt(mux.Vars(r)["UserID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPUsersConnectedRequest 2 : ", err.Error())
		return nil, err
	}
	(&request).UserID = userID

	friendID, err := strconv.ParseInt(mux.Vars(r)["FriendID"], 10, 64)
	if err != nil {
		fmt.Println("Error DecodeHTTPUsersConnectedRequest 2 : ", err.Error())
		return nil, err
	}
	(&request).FriendID = friendID

	return request, nil
}

func DecodeHTTPUsersConnectedResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response usersConnectedResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPUsersConnectedResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func UsersConnectedHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/friends/{UserID:[0-9]+}/connected/{FriendID:[0-9]+}").Handler(httptransport.NewServer(
		endpoints.UsersConnectedEndpoint,
		DecodeHTTPUsersConnectedRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "UsersConnected", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) UsersConnected(ctx context.Context, userID, friendID int64) (bool, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "usersConnected",
			"userID", userID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.UsersConnected(ctx, userID, friendID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) UsersConnected(ctx context.Context, userID, friendID int64) (bool, error) {
	connected, err := mw.next.UsersConnected(ctx, userID, friendID)
	mw.ints.Add(1)
	return connected, err
}

/*************** Main ***************/
/* Main */
func BuildUsersConnectedEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var gefmEndpoint endpoint.Endpoint
	{
		gefmDuration := duration.With("method", "UsersConnected")
		gefmLogger := log.With(logger, "method", "UsersConnected")

		gefmEndpoint = UsersConnectedEndpoint(svc)
		gefmEndpoint = opentracing.TraceServer(tracer, "UsersConnected")(gefmEndpoint)
		gefmEndpoint = EndpointLoggingMiddleware(gefmLogger)(gefmEndpoint)
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) UsersConnected(ctx context.Context, userID, friendID int64) (bool, error) {
	request := usersConnectedRequest{UserID: userID, FriendID: friendID}
	response, err := e.UsersConnectedEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error UsersConnected : ", err.Error())
		return false, err
	}
	return response.(usersConnectedResponse).Connected, str2err(response.(usersConnectedResponse).Err)
}

func ClientUsersConnected(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/friends/{UserID:[0-9]+}/connected/{FriendID:[0-9]+}"),
		EncodeHTTPGenericRequest,
		DecodeHTTPUsersConnectedResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "UsersConnected")(gefmEndpoint)
	return gefmEndpoint, nil
}
