package svcdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/johnnadratowski/golang-neo4j-bolt-driver"

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
func (s Service) AddSuccess(c context.Context, userID int64, successValue string) error {
	var stmt golangNeo4jBoltDriver.Stmt

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("AddSuccess (WaitConnection) : " + err.Error())
		return err
	}
	defer CloseConnection(conn)

	stmt, err = conn.PrepareNeo(`
		MATCH (s:SUCCESS) WHERE s.Value = {value}
		MATCH (u:USER) WHERE ID(u) = {id}
		CREATE (u)-[g:GOT]->(s)
		SET u.SuccessPoints = u.SuccessPoints+10
		RETURN u, g, s
	`)

	defer stmt.Close()

	if err != nil {
		fmt.Println("AddSuccess (PrepareNeo) : " + err.Error())
		return err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id":    userID,
		"value": successValue,
	})

	if err != nil {
		fmt.Println("AddSuccess (QueryNeo) : " + err.Error())
		return err
	}

	_, _, err = rows.NextNeo()
	if err != nil {
		fmt.Println("AddSuccess (NextNeo) : " + err.Error())
		return err
	}

	return err
}

/*************** Endpoint ***************/
type AddSuccessRequest struct {
	UserID  int64  `json:"userID"`
	Success string `json:"success"`
}

type AddSuccessResponse struct {
	Err string `json:"err,omitempty"`
}

func AddSuccessEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AddSuccessRequest)

		err := svc.AddSuccess(ctx, req.UserID, req.Success)
		if err != nil {
			fmt.Println("AddSuccessEndpoint : " + err.Error())
			return AddSuccessResponse{Err: err.Error()}, nil
		}

		return AddSuccessResponse{}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPAddSuccessRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request AddSuccessRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func DecodeHTTPAddSuccessResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response AddSuccessResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func AddSuccessHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/success").Handler(httptransport.NewServer(
		endpoints.AddSuccessEndpoint,
		DecodeHTTPAddSuccessRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "AddSuccess", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) AddSuccess(ctx context.Context, userID int64, successValue string) error {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "AddSuccess",
			"userID", userID,
			"success", successValue,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.AddSuccess(ctx, userID, successValue)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) AddSuccess(ctx context.Context, userID int64, successValue string) error {
	err := mw.next.AddSuccess(ctx, userID, successValue)
	mw.ints.Add(1)
	return err
}

/*************** Main ***************/
/* Main */
func BuildAddSuccessEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "AddSuccess")
		csLogger := log.With(logger, "method", "AddSuccess")

		csEndpoint = AddSuccessEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "AddSuccess")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
func (e Endpoints) AddSuccess(ctx context.Context, userID int64, successValue string) error {
	request := AddSuccessRequest{UserID: userID, Success: successValue}
	response, err := e.AddSuccessEndpoint(ctx, request)
	if err != nil {
		return err
	}
	return str2err(response.(AddSuccessResponse).Err)
}

func ClientAddSuccess(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, "/success"),
		EncodeHTTPGenericRequest,
		DecodeHTTPAddSuccessResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "AddSuccess")(ceEndpoint)
	return ceEndpoint, nil
}
