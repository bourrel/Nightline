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
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"
)

/*************** Service ***************/
func (s Service) FailOrder(ctx context.Context, orderID int64) (Order, error) {
    var order Order

    conn, err := WaitConnection(5)
    if err != nil {
		fmt.Println("FailOrder (WaitConnection) : " + err.Error())
		return order, err
	}
    defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
    MATCH (o:ORDER)-[DONE]-(s:STEP)
    WHERE ID(o) = {orderID} AND (NOT EXISTS(s.Result) OR s.Result = "")
    SET o.Done = "false", s.Result = "false"
    RETURN ID(o)`)
	defer stmt.Close()

    if err != nil {
		fmt.Println("FailOrder (PrepareNeo) : " + err.Error())
		return order, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"orderID":     orderID,
	})

	if err != nil {
		fmt.Println("FailOrder (QueryNeo) : " + err.Error())
		return order, err
	}

	data, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("FailOrder (NextNeo) : " + err.Error())
		return order, err
	}

	orderID = data[0].(int64)

	order, err = s.GetOrder(ctx, orderID)
	if err != nil {
		fmt.Println("FailOrder (GetOrder) : " + err.Error())
		return order, err
	}
	return order, err
}

/*************** Endpoint ***************/
type failOrderRequest struct {
	OrderID	int64 `json:"order"`
}

type failOrderResponse struct {
	Order	Order  `json:"order"`
	Err		string `json:"err,omitempty"`
}

func FailOrderEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(failOrderRequest)
		order, err := svc.FailOrder(ctx, req.OrderID)

		// Create node
		if err != nil {
			fmt.Println("Error FailOrderEndpoint 1 : ", err.Error())
			return failOrderResponse{Order: order, Err: err.Error()}, nil
		}

		return failOrderResponse{Order: order, Err: ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPFailOrderRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request failOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		fmt.Println("Error DecodeHTTPFailOrderRequest : ", err.Error(), err)
		return nil, err
	}
	return request, nil
}

func DecodeHTTPFailOrderResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response failOrderResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPFailOrderResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func FailOrderHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/order/fail").Handler(httptransport.NewServer(
		endpoints.FailOrderEndpoint,
		DecodeHTTPFailOrderRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "FailOrder", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) FailOrder(ctx context.Context, orderID int64) (Order, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "failOrder",
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.FailOrder(ctx, orderID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) FailOrder(ctx context.Context, orderID int64) (Order, error) {
	v, err := mw.next.FailOrder(ctx, orderID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildFailOrderEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "FailOrder")
		csLogger := log.With(logger, "method", "FailOrder")

		csEndpoint = FailOrderEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "FailOrder")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
func (e Endpoints) FailOrder(ctx context.Context, orderID int64) (Order, error) {
	request := failOrderRequest{OrderID: orderID}
	response, err := e.FailOrderEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error FailOrder : ", err.Error())
		return response.(failOrderResponse).Order, err
	}
	return response.(failOrderResponse).Order, str2err(response.(failOrderResponse).Err)
}

func ClientFailOrder(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, "/order/fail"),
		EncodeHTTPGenericRequest,
		DecodeHTTPFailOrderResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "FailOrder")(ceEndpoint)
	return ceEndpoint, nil
}
