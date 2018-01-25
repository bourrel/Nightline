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
func (s Service) UserLeaveSoiree(_ context.Context, userID int64, soireeID int64) (Soiree, bool, error) {
	var soiree Soiree

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("UserLeaveSoiree (WaitConnection) : " + err.Error())
		return soiree, false, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (u:USER), (s:SOIREE) WHERE ID(u) = {uid} AND ID(s) = {sid}
		CREATE (u)-[:LEAVE {When: {date}} ]->(s)
		RETURN u, s
	`)
	defer stmt.Close()
	if err != nil {
		fmt.Println("UserLeaveSoiree (PrepareNeo) : " + err.Error())
		return soiree, false, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"uid":  userID,
		"sid":  soireeID,
		"date": time.Now().Format(timeForm),
	})

	if err != nil {
		fmt.Println("UserLeaveSoiree (QueryNeo) : " + err.Error())
		return soiree, false, err
	}

	row, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("UserLeaveSoiree (NextNeo) : " + err.Error())
		return soiree, false, err
	}

	(&soiree).NodeToSoiree(row[1].(graph.Node))
	return soiree, true, nil
}

/*************** Endpoint ***************/
type userLeaveSoireeRequest struct {
	UserID   int64 `json:"uid"`
	SoireeID int64 `json:"sid"`
}

type userLeaveSoireeResponse struct {
	Soiree  Soiree `json:"soiree"`
	Present bool   `json:"present"`
	Err     string `json:"err,omitempty"`
}

func UserLeaveSoireeEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(userLeaveSoireeRequest)
		soiree, present, err := svc.UserLeaveSoiree(ctx, req.UserID, req.SoireeID)
		if err != nil {
			fmt.Println("Error UserLeaveSoireeEndpoint 1 " + err.Error())
			return userLeaveSoireeResponse{soiree, present, err.Error()}, nil
		}

		soiree.Menu, err = svc.GetMenuFromSoiree(ctx, soiree.ID)
		if err != nil {
			fmt.Println("Error UserLeaveSoireeEndpoint 2 : ", err.Error())
			return userLeaveSoireeResponse{soiree, present, err.Error()}, nil
		}

		return userLeaveSoireeResponse{soiree, present, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPUserLeaveSoireeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request userLeaveSoireeRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		fmt.Println("Error DecodeHTTPUserLeaveSoireeRequest : " + err.Error())
		return nil, err
	}

	return request, nil
}

func DecodeHTTPUserLeaveSoireeResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response userLeaveSoireeResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPUserLeaveSoireeResponse : " + err.Error())
		return nil, err
	}
	return response, nil
}

func UserLeaveSoireeHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/userLeaveSoiree").Handler(httptransport.NewServer(
		endpoints.UserLeaveSoireeEndpoint,
		DecodeHTTPUserLeaveSoireeRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "UserLeaveSoiree", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) UserLeaveSoiree(ctx context.Context, userID, soireeID int64) (Soiree, bool, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "userLeaveSoiree",
			"userID", userID,
			"soireeID", soireeID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.UserLeaveSoiree(ctx, userID, soireeID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) UserLeaveSoiree(ctx context.Context, userID, soireeID int64) (Soiree, bool, error) {
	soiree, present, err := mw.next.UserLeaveSoiree(ctx, userID, soireeID)
	mw.ints.Add(1)
	return soiree, present, err
}

/*************** Main ***************/
/* Main */
func BuildUserLeaveSoireeEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var gefmEndpoint endpoint.Endpoint
	{
		gefmDuration := duration.With("method", "UserLeaveSoiree")
		gefmLogger := log.With(logger, "method", "UserLeaveSoiree")

		gefmEndpoint = UserLeaveSoireeEndpoint(svc)
		gefmEndpoint = opentracing.TraceServer(tracer, "UserLeaveSoiree")(gefmEndpoint)
		gefmEndpoint = EndpointLoggingMiddleware(gefmLogger)(gefmEndpoint)
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) UserLeaveSoiree(ctx context.Context, userID, soireeID int64) (Soiree, bool, error) {
	var s Soiree

	request := userLeaveSoireeRequest{UserID: userID, SoireeID: soireeID}
	response, err := e.UserLeaveSoireeEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error UserLeaveSoiree" + err.Error())
		return s, false, err
	}
	s = response.(userLeaveSoireeResponse).Soiree
	return s, response.(userLeaveSoireeResponse).Present, str2err(response.(userLeaveSoireeResponse).Err)
}

func ClientUserLeaveSoiree(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/userLeaveSoiree"),
		EncodeHTTPGenericRequest,
		DecodeHTTPUserLeaveSoireeResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "UserLeaveSoiree")(gefmEndpoint)
	return gefmEndpoint, nil
}
