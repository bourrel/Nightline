package svcdb

import (
	"context"
	"encoding/json"
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
func (s Service) UserJoinSoiree(_ context.Context, userID int64, soireeID int64) (Soiree, bool, error) {
	var soiree Soiree

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("UserJoinSoiree (WaitConnection) : " + err.Error())
		return soiree, false, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (u:USER), (s:SOIREE) WHERE ID(u) = {uid} AND ID(s) = {sid}
		CREATE (u)-[:JOIN {When: {date}} ]->(s)
		RETURN u, s
	`)
	defer stmt.Close()
	if err != nil {
		fmt.Println("UserJoinSoiree (PrepareNeo) : " + err.Error())
		return soiree, false, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"uid":  userID,
		"sid":  soireeID,
		"date": time.Now().Format(timeForm),
	})

	if err != nil {
		fmt.Println("UserJoinSoiree (QueryNeo) : " + err.Error())
		return soiree, false, err
	}

	row, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("UserJoinSoiree (NextNeo) : " + err.Error())
		return soiree, false, err
	}

	(&soiree).NodeToSoiree(row[1].(graph.Node))
	return soiree, true, nil
}

/*************** Endpoint ***************/
type userJoinSoireeRequest struct {
	UserID   int64 `json:"uid"`
	SoireeID int64 `json:"sid"`
}

type userJoinSoireeResponse struct {
	Soiree  Soiree `json:"soiree"`
	Present bool   `json:"present"`
	Err     string `json:"err,omitempty"`
}

func UserJoinSoireeEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(userJoinSoireeRequest)
		soiree, present, err := svc.UserJoinSoiree(ctx, req.UserID, req.SoireeID)
		if err != nil {
			fmt.Println("Error UserJoinSoireeEndpoint 1 : " + err.Error())
			return userJoinSoireeResponse{soiree, present, err.Error()}, nil
		}

		soiree.Menu, err = svc.GetMenuFromSoiree(ctx, soiree.ID)
		if err != nil {
			fmt.Println("Error UserJoinSoireeEndpoint 2 : ", err.Error())
			return userJoinSoireeResponse{soiree, present, err.Error()}, nil
		}

		soiree.Friends, err = svc.GetConnectedFriends(ctx, soiree.ID)
		if err != nil {
			fmt.Println("Error UserJoinSoireeEndpoint 3 : ", err.Error())
			return userJoinSoireeResponse{soiree, present, err.Error()}, nil
		}

		return userJoinSoireeResponse{soiree, present, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPUserJoinSoireeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request userJoinSoireeRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		fmt.Println("Error DecodeHTTPUserJoinSoireeRequest : " + err.Error())
		return nil, err
	}

	return request, nil
}

func DecodeHTTPUserJoinSoireeResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response userJoinSoireeResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPUserJoinSoireeResponse : " + err.Error())
		return nil, err
	}
	return response, nil
}

func UserJoinSoireeHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/userJoinSoiree").Handler(httptransport.NewServer(
		endpoints.UserJoinSoireeEndpoint,
		DecodeHTTPUserJoinSoireeRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "UserJoinSoiree", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) UserJoinSoiree(ctx context.Context, userID, soireeID int64) (Soiree, bool, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "userJoinSoiree",
			"userID", userID,
			"soireeID", soireeID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.UserJoinSoiree(ctx, userID, soireeID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) UserJoinSoiree(ctx context.Context, userID, soireeID int64) (Soiree, bool, error) {
	soiree, present, err := mw.next.UserJoinSoiree(ctx, userID, soireeID)
	mw.ints.Add(1)
	return soiree, present, err
}

/*************** Main ***************/
/* Main */
func BuildUserJoinSoireeEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var gefmEndpoint endpoint.Endpoint
	{
		gefmDuration := duration.With("method", "UserJoinSoiree")
		gefmLogger := log.With(logger, "method", "UserJoinSoiree")

		gefmEndpoint = UserJoinSoireeEndpoint(svc)
		gefmEndpoint = opentracing.TraceServer(tracer, "UserJoinSoiree")(gefmEndpoint)
		gefmEndpoint = EndpointLoggingMiddleware(gefmLogger)(gefmEndpoint)
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) UserJoinSoiree(ctx context.Context, userID, soireeID int64) (Soiree, bool, error) {
	var s Soiree

	request := userJoinSoireeRequest{UserID: userID, SoireeID: soireeID}
	response, err := e.UserJoinSoireeEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error UserJoinSoiree" + err.Error())
		return s, false, err
	}
	s = response.(userJoinSoireeResponse).Soiree
	return s, response.(userJoinSoireeResponse).Present, str2err(response.(userJoinSoireeResponse).Err)
}

func ClientUserJoinSoiree(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/userJoinSoiree"),
		EncodeHTTPGenericRequest,
		DecodeHTTPUserJoinSoireeResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "UserJoinSoiree")(gefmEndpoint)
	return gefmEndpoint, nil
}
