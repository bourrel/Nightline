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
	"github.com/johnnadratowski/golang-neo4j-bolt-driver/structures/graph"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"
)

/*************** Service ***************/
func (s Service) GetConsoByOrderID(_ context.Context, orderID int64) (Conso, error) {
	var conso Conso

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetConsoByOrderID (WaitConnection) : " + err.Error())
		return conso, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
	MATCH (o:ORDER)-[:FOR]->(c:CONSO)
	WHERE ID(o) = {id}
	RETURN c`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("GetConsoByOrderID (PrepareNeo) : " + err.Error())
		return conso, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": orderID,
	})
	if err != nil {
		fmt.Println("GetConsoByOrderID (QueryNeo) : " + err.Error())
		return conso, err
	}

	row, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("GetConsoByOrderID (NextNeo) : " + err.Error())
		return conso, err
	}

	(&conso).NodeToConso(row[0].(graph.Node))
	return conso, nil
}

/*************** Endpoint ***************/
type getConsoByOrderIDRequest struct {
	ID int64 `json:"id"`
}

type getConsoByOrderIDResponse struct {
	Conso Conso  `json:"conso"`
	Err   string `json:"err,omitempty"`
}

func GetConsoByOrderIDEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getConsoByOrderIDRequest)
		conso, err := svc.GetConsoByOrderID(ctx, req.ID)
		if err != nil {
			return getConsoByOrderIDResponse{conso, err.Error()}, nil
		}
		return getConsoByOrderIDResponse{conso, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetConsoByOrderIDRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getConsoByOrderIDRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	consoiD, err := strconv.ParseInt(mux.Vars(r)["orderID"], 10, 64)
	if err != nil {
		return nil, err
	}
	(&request).ID = consoiD

	return request, nil
}

func DecodeHTTPGetConsoByOrderIDResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getConsoByOrderIDResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetConsoByOrderIDHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/orders/{orderID}/conso").Handler(httptransport.NewServer(
		endpoints.GetConsoByOrderIDEndpoint,
		DecodeHTTPGetConsoByOrderIDRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetConsoByOrderID", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetConsoByOrderID(ctx context.Context, orderID int64) (Conso, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getConsoByOrderID",
			"orderID", orderID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetConsoByOrderID(ctx, orderID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetConsoByOrderID(ctx context.Context, orderID int64) (Conso, error) {
	v, err := mw.next.GetConsoByOrderID(ctx, orderID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetConsoByOrderIDEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetConsoByOrderID")
		csLogger := log.With(logger, "method", "GetConsoByOrderID")

		csEndpoint = GetConsoByOrderIDEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetConsoByOrderID")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetConsoByOrderID(ctx context.Context, ID int64) (Conso, error) {
	var conso Conso

	request := getConsoByOrderIDRequest{ID: ID}
	response, err := e.GetConsoByOrderIDEndpoint(ctx, request)
	if err != nil {
		return conso, err
	}
	conso = response.(getConsoByOrderIDResponse).Conso
	return conso, str2err(response.(getConsoByOrderIDResponse).Err)
}

func EncodeHTTPGetConsoByOrderIDRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(getConsoByOrderIDRequest).ID)
	encodedUrl, err := route.Path(r.URL.Path).URL("orderID", id)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetConsoByOrderID(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/orders/{orderID}/conso"),
		EncodeHTTPGetConsoByOrderIDRequest,
		DecodeHTTPGetConsoByOrderIDResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "GetConsoByOrderID")(gefmEndpoint)
	return gefmEndpoint, nil
}
