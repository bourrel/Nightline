package svcdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
	"io"

	"github.com/go-kit/kit/tracing/opentracing"
	mux "github.com/gorilla/mux"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"
)

/*************** Service ***************/
func (s Service) SearchOrders(ctx context.Context, order Order) (Orders, error) {
	var orders Orders
    var req, reqMatch, reqWhere string

	i := 0
	args := make(map[string]interface{})

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("SearchOrders (WaitConnection) : " + err.Error())
		return orders, err
	}

	/* Init */
	reqMatch = `MATCH (o:ORDER)`
	if len(order.Done) > 0 || len(order.Users) > 0 || len(order.Steps) > 0 || order.Soiree.ID > 0 {
		reqWhere = `WHERE ID(o) > 0`
	} else {
		reqWhere = ``
	}

	/* Order Done */
	if len(order.Done) > 0 {
		if order.Done == "set" {
			reqWhere += ` AND EXISTS(o.Done)`
		} else if order.Done == "unset" {
			reqWhere += ` AND NOT EXISTS(o.Done)`
		} else {
			reqWhere += ` AND o.Done = {odone}`
			args["odone"] = order.Done
		}
	}

	/* Soiree */
	if order.Soiree.ID > 0 {
		reqMatch += `, (o)-->(so:SOIREE)`
		reqWhere += ` AND ID(so) = {soid}`
		args["soid"] = order.Soiree.ID
	}

	/* Users */
	for _, user := range order.Users {
		nodeName := `u` + strconv.Itoa(i)
		relName := nodeName + `r`
        reqMatch += `, (o)-[` + relName + `:TO]->(` + nodeName + `:USER)`
		reqWhere += ` AND ID(` + nodeName + `) = {` + nodeName + `id}`
        args[nodeName + `id`] = user.User.ID
		if len(user.Approved) > 0 {
			reqWhere += ` AND ` + relName + `.Approved = {` + relName + `approved}`
			args[relName + `approved`] = user.Approved
		}
		i++
	}

	/* Steps */
	for _, step := range order.Steps {
		nodeName := `st` + strconv.Itoa(i)
		relName := nodeName + `r`
        reqMatch += `, (o)-[` + relName + `:DONE]->(` + nodeName + `:STEP)`
		reqWhere += ` AND ` + nodeName + `.Name = {` + nodeName + `name}`
        args[nodeName + `name`] = step.Name
		if len(step.Result) > 0 {
			reqWhere += ` AND ` + nodeName + `.Result = {` + nodeName + `result}`
			args[nodeName + `result`] = step.Result
		}
		i++
	}

	/* Final */
	req = reqMatch + ` ` + reqWhere + ` RETURN ID(o)`
	stmt, err := conn.PrepareNeo(req)
	fmt.Println("SearchOrders (request) : " + req)

	if err != nil {
		fmt.Println("SearchOrders (PrepareNeo) : " + err.Error())
		return orders, err
	}

	rows, err := stmt.QueryNeo(args)
	if err != nil {
		fmt.Println("SearchOrders (QueryNeo) : " + err.Error())
		return orders, err
	}

	row, _, err := rows.NextNeo()
	for row != nil && err == nil {
		if err != nil && err != io.EOF {
			fmt.Println("SearchOrders (NextNeo) : " + err.Error())
			return orders, err
		} else if err != io.EOF {
			tmpOrder, _ := s.GetOrder(ctx, row[0].(int64))
			orders = append(orders, tmpOrder)
		}
		row, _, err = rows.NextNeo()
	}

	stmt.Close()
	CloseConnection(conn)

	return orders, nil
}

/*************** Endpoint ***************/
type searchOrdersRequest struct {
	Order		Order	`json:"order"`
}

type searchOrdersResponse struct {
	Orders		Orders	`json:"orders"`
	Err			string	`json:"err,omitempty"`
}

func SearchOrdersEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(searchOrdersRequest)
		orders, err := svc.SearchOrders(ctx, req.Order)
		if err != nil {
			fmt.Println("Error SearchOrdersEndpoint : ", err.Error())
			return searchOrdersResponse{orders, err.Error()}, nil
		}
		return searchOrdersResponse{orders, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPSearchOrdersRequest(_ context.Context, r *http.Request) (interface{}, error) {
    var request searchOrdersRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		fmt.Println("Error DecodeHTTPSearchOrdersRequest : ", err.Error(), err)
		return nil, err
	}
	return request, nil
}

func DecodeHTTPSearchOrdersResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response searchOrdersResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPSearchOrdersResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func SearchOrdersHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/search/orders").Handler(httptransport.NewServer(
		endpoints.SearchOrdersEndpoint,
		DecodeHTTPSearchOrdersRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "SearchOrders", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) SearchOrders(ctx context.Context, order Order) (Orders, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "searchOrders",
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.SearchOrders(ctx, order)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) SearchOrders(ctx context.Context, order Order) (Orders, error) {
	v, err := mw.next.SearchOrders(ctx, order)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildSearchOrdersEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "SearchOrders")
		csLogger := log.With(logger, "method", "SearchOrders")

		csEndpoint = SearchOrdersEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "SearchOrders")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forsearch limiter & circuitbreaker for now kthx
func (e Endpoints) SearchOrders(ctx context.Context, order Order) (Orders, error) {
	var orders Orders

	request := searchOrdersRequest{Order: order}
	response, err := e.SearchOrdersEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error Client SearchOrders : ", err.Error())
		return orders, err
	}
	orders = response.(searchOrdersResponse).Orders
	return orders, str2err(response.(searchOrdersResponse).Err)
}

func ClientSearchOrders(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, "/search/orders"),
		EncodeHTTPGenericRequest,
		DecodeHTTPSearchOrdersResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "SearchOrders")(ceEndpoint)
	return ceEndpoint, nil
}
