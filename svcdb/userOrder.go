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
func (s Service) UserOrder(_ context.Context, user User, soiree Soiree, conso Conso) (int64, error) {

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("UserOrder (WaitConnection) : " + err.Error())
		return 0, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (u:USER), (s:SOIREE), (c:CONSO)
		WHERE ID(u) = {uid} AND ID(s) = {sid} AND ID(c) = {cid}
		CREATE (o:ORDER {
			Price: {price},
			Created: {created},
			Reference: {reference},
			Paid: {paid},
			Delivered: {delivered},
			Completed: {completed}
		}),
		(u)-[:ORDERED]->(o),
		(o)-[:DURING]->(s),
		(o)-[:PURCHASED]->(c)
		RETURN o
	`)
	defer stmt.Close()
	if err != nil {
		fmt.Println("UserOrder (PrepareNeo) : " + err.Error())
		return 0, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"uid":       user.ID,
		"sid":       soiree.ID,
		"cid":       conso.ID,
		"price":     conso.Price,
		"created":   time.Now().Format(timeForm),
		"reference": "",
		"paid":      nil,
		"delivered": nil,
		"completed": nil,
	})

	if err != nil {
		fmt.Println("UserOrder (QueryNeo) : " + err.Error())
		return 0, err
	}

	row, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("UserOrder (NextNeo) : " + err.Error())
		return 0, err
	}

	orderID := row[0].(graph.Node).NodeIdentity
	return orderID, nil
}

/*************** Endpoint ***************/
type userOrderRequest struct {
	User   User   `json:"user"`
	Soiree Soiree `json:"soiree"`
	Conso  Conso  `json:"conso"`
}

type userOrderResponse struct {
	OrderID int64  `json:"orderid"`
	Err     string `json:"err,omitempty"`
}

func UserOrderEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(userOrderRequest)
		orderID, err := svc.UserOrder(ctx, req.User, req.Soiree, req.Conso)
		if err != nil {
			fmt.Println("Error UserOrderEndpoint" + err.Error())
			return userOrderResponse{orderID, err.Error()}, nil
		}
		return userOrderResponse{orderID, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPUserOrderRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request userOrderRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		fmt.Println("Error DecodeHTTPUserOrderRequest : " + err.Error())
		return nil, err
	}

	return request, nil
}

func DecodeHTTPUserOrderResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response userOrderResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPUserOrderResponse : " + err.Error())
		return nil, err
	}
	return response, nil
}

func UserOrderHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/userOrder").Handler(httptransport.NewServer(
		endpoints.UserOrderEndpoint,
		DecodeHTTPUserOrderRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "UserOrder", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) UserOrder(ctx context.Context, user User, soiree Soiree, conso Conso) (int64, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "userOrder",
			"userID", user.ID,
			"soireeID", soiree.ID,
			"consoID", conso.ID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.UserOrder(ctx, user, soiree, conso)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) UserOrder(ctx context.Context, user User, soiree Soiree, conso Conso) (int64, error) {
	status, err := mw.next.UserOrder(ctx, user, soiree, conso)
	mw.ints.Add(1)
	return status, err
}

/*************** Main ***************/
/* Main */
func BuildUserOrderEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var gefmEndpoint endpoint.Endpoint
	{
		gefmDuration := duration.With("method", "UserOrder")
		gefmLogger := log.With(logger, "method", "UserOrder")

		gefmEndpoint = UserOrderEndpoint(svc)
		gefmEndpoint = opentracing.TraceServer(tracer, "UserOrder")(gefmEndpoint)
		gefmEndpoint = EndpointLoggingMiddleware(gefmLogger)(gefmEndpoint)
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) UserOrder(ctx context.Context, user User, soiree Soiree, conso Conso) (int64, error) {
	request := userOrderRequest{User: user, Soiree: soiree, Conso: conso}
	response, err := e.UserOrderEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error UserOrder" + err.Error())
		return response.(userOrderResponse).OrderID, err
	}
	return response.(userOrderResponse).OrderID, str2err(response.(userOrderResponse).Err)
}

func ClientUserOrder(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/userOrder"),
		EncodeHTTPGenericRequest,
		DecodeHTTPUserOrderResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "UserOrder")(gefmEndpoint)
	return gefmEndpoint, nil
}
