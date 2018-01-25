package svcdb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
func (s Service) GetSoireeOrders(_ context.Context, soireeID int64) ([]Order, error) {
	var orders []Order
	var tmpOrder Order

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetSoireeOrders (WaitConnection) : " + err.Error())
		return nil, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`MATCH (o:ORDER)-[:DURING]->(s:SOIREE) WHERE ID(s) = {id} RETURN o`)
	defer stmt.Close()
	if err != nil {
		fmt.Println("GetSoireeOrders (PrepareNeo) : " + err.Error())
		return orders, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": soireeID,
	})

	if err != nil {
		fmt.Println("GetSoireeOrders (QueryNeo) : " + err.Error())
		return orders, err
	}

	row, _, err := rows.NextNeo()
	for row != nil && err == nil {
		if err != nil && err != io.EOF {
			fmt.Println("GetSoireeOrders (---) : " + err.Error())
			panic(err)
		} else if err != io.EOF {
			(&tmpOrder).NodeToOrder(row[0].(graph.Node))
			orders = append(orders, tmpOrder)
		}
		row, _, err = rows.NextNeo()
	}

	return orders, nil
}

/*************** Endpoint ***************/
type getSoireeOrdersRequest struct {
	SoireeID int64 `json:"id"`
}

type getSoireeOrdersResponse struct {
	Orders []Order `json:"orders"`
	Err    string  `json:"err,omitempty"`
}

func GetSoireeOrdersEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getSoireeOrdersRequest)
		orders, err := svc.GetSoireeOrders(ctx, req.SoireeID)
		if err != nil {
			return getSoireeOrdersResponse{Orders: orders, Err: err.Error()}, nil
		}
		return getSoireeOrdersResponse{Orders: orders, Err: ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetSoireeOrdersRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getSoireeOrdersRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	soireeID, err := strconv.ParseInt(mux.Vars(r)["SoireeID"], 10, 64)
	if err != nil {
		return nil, err
	}
	(&request).SoireeID = soireeID

	return request, nil
}

func DecodeHTTPGetSoireeOrdersResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getSoireeOrdersResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetSoireeOrdersHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/soirees/{SoireeID}/orders").Handler(httptransport.NewServer(
		endpoints.GetSoireeOrdersEndpoint,
		DecodeHTTPGetSoireeOrdersRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetSoireeOrders", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetSoireeOrders(ctx context.Context, soireeID int64) ([]Order, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getSoireeOrders",
			"soireeID", soireeID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetSoireeOrders(ctx, soireeID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetSoireeOrders(ctx context.Context, soireeID int64) ([]Order, error) {
	v, err := mw.next.GetSoireeOrders(ctx, soireeID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetSoireeOrdersEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var gefmEndpoint endpoint.Endpoint
	{
		gefmDuration := duration.With("method", "GetSoireeOrders")
		gefmLogger := log.With(logger, "method", "GetSoireeOrders")

		gefmEndpoint = GetSoireeOrdersEndpoint(svc)
		gefmEndpoint = opentracing.TraceServer(tracer, "GetSoireeOrders")(gefmEndpoint)
		gefmEndpoint = EndpointLoggingMiddleware(gefmLogger)(gefmEndpoint)
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetSoireeOrders(ctx context.Context, soireeID int64) ([]Order, error) {
	var s []Order

	request := getSoireeOrdersRequest{SoireeID: soireeID}
	response, err := e.GetSoireeOrdersEndpoint(ctx, request)
	if err != nil {
		return s, err
	}
	s = response.(getSoireeOrdersResponse).Orders
	return s, str2err(response.(getSoireeOrdersResponse).Err)
}

func EncodeHTTPGetSoireeOrdersRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(getSoireeOrdersRequest).SoireeID)
	encodedUrl, err := route.Path(r.URL.Path).URL("SoireeID", id)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetSoireeOrders(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/soirees/{SoireeID}/orders"),
		EncodeHTTPGetSoireeOrdersRequest,
		DecodeHTTPGetSoireeOrdersResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "GetSoireeOrders")(gefmEndpoint)
	return gefmEndpoint, nil
}
