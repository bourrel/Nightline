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
func (s Service) GetOrdersBySoiree(_ context.Context, soireeID int64) ([]Order, error) {
	var orders []Order

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetOrdersBySoiree (WaitConnection) : " + err.Error())
		return nil, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo("MATCH (n:ORDER)-->(u:SOIREE) WHERE ID(u) = {id} RETURN n")
	if err != nil {
		fmt.Println("GetAllOrders (PrepareNeo)")
		panic(err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": soireeID,
	})
	if err != nil {
		fmt.Println("GetAllOrders (QueryNeo)")
		panic(err)
	}

	// Here we loop through the rows until we get the metadata object
	// back, meaning the row stream has been fully consumed
	var tmpOrder Order

	row, _, err := rows.NextNeo()
	for row != nil && err == nil {

		if err != nil && err != io.EOF {
			fmt.Println("GetAllOrders (---)")
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
type getOrdersBySoireeRequest struct {
	SoireeID int64 `json:"id"`
}

type getOrdersBySoireeResponse struct {
	Orders []Order `json:"orders"`
	Err    string  `json:"err,omitempty"`
}

func GetOrdersBySoireeEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getOrdersBySoireeRequest)
		soirees, err := svc.GetOrdersBySoiree(ctx, req.SoireeID)
		if err != nil {
			return getOrdersBySoireeResponse{soirees, err.Error()}, nil
		}
		return getOrdersBySoireeResponse{soirees, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetOrdersBySoireeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getOrdersBySoireeRequest

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

func DecodeHTTPGetOrdersBySoireeResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getOrdersBySoireeResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetOrdersBySoireeHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/ordersBySoiree/{SoireeID}").Handler(httptransport.NewServer(
		endpoints.GetOrdersBySoireeEndpoint,
		DecodeHTTPGetOrdersBySoireeRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetOrdersBySoiree", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetOrdersBySoiree(ctx context.Context, soireeID int64) ([]Order, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getOrdersBySoiree",
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetOrdersBySoiree(ctx, soireeID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetOrdersBySoiree(ctx context.Context, soireeID int64) ([]Order, error) {
	v, err := mw.next.GetOrdersBySoiree(ctx, soireeID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetOrdersBySoireeEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetOrdersBySoiree")
		csLogger := log.With(logger, "method", "GetOrdersBySoiree")

		csEndpoint = GetOrdersBySoireeEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetOrdersBySoiree")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetOrdersBySoiree(ctx context.Context, soireeID int64) ([]Order, error) {
	var et []Order

	request := getOrdersBySoireeRequest{SoireeID: soireeID}
	response, err := e.GetOrdersBySoireeEndpoint(ctx, request)
	if err != nil {
		return et, err
	}
	et = response.(getOrdersBySoireeResponse).Orders
	return et, str2err(response.(getOrdersBySoireeResponse).Err)
}

func EncodeHTTPGetOrdersBySoireeRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(getOrdersBySoireeRequest).SoireeID)
	encodedUrl, err := route.Path(r.URL.Path).URL("SoireeID", id)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetOrdersBySoiree(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/ordersBySoiree/{SoireeID}"),
		EncodeHTTPGetOrdersBySoireeRequest,
		DecodeHTTPGetOrdersBySoireeResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "GetOrdersBySoiree")(ceEndpoint)
	return ceEndpoint, nil
}
