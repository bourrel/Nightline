package svcdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"errors"

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
func (s Service) CreateOrder(ctx context.Context, o Order) (Order, error) {
	var order Order
	var req, reqMatch, reqWhere, reqCreate string
	var sumPrice int64

	i := 0
	args := make(map[string]interface{})

	// soiree, err := s.GetSoireeByID(

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("CreateOrder (WaitConnection) : " + err.Error())
		return order, err
	}
	defer CloseConnection(conn)

	/* Init */
	reqMatch = `MATCH (so:SOIREE)`
	reqWhere = `WHERE ID(so) = {soid}`
	reqCreate = `
        CREATE (o:ORDER {Price: {price}}), (st:STEP {Name: {stname}, Date: {stdate}}),
        (o)-[:DURING]->(so),
        (o)-[:DONE]->(st)
    `
	args["price"] = o.Price
	args["soid"] = o.Soiree.ID
	args["stname"] = "Issued"
	args["stdate"] = time.Now().Format(timeForm)

	/* Users */
	for _, user := range o.Users {
		nodeName := `u` + strconv.Itoa(i)
		relName := nodeName + `r`
		reqMatch += `, (` + nodeName + `:USER)`
		reqWhere += ` AND ID(` + nodeName + `) = {` + nodeName + `id}`
		reqCreate += `, (o)-[` + relName + `:TO` +
			`{Price: {` + relName + `price}, Reference: {` + relName + `reference}, Approved: ""}]` +
			`->(`  + nodeName + `)`
		args[nodeName + `id`] = user.User.ID
		args[relName + `price`] = user.Price
		args[relName + `reference`] = user.Reference
		sumPrice += user.Price
		i++
	}

	/* Consos */
	for _, conso := range o.Consos {
		nodeName := `c` + strconv.Itoa(i)
		relName := nodeName + `r`
		reqMatch += `, (` + nodeName + `:CONSO)`
		reqWhere += ` AND ID(` + nodeName + `) = {` + nodeName + `id}`
		reqCreate += `, (o)-[` + relName + `:FOR` +
			`{Amount: {` + relName + `amount}}]` +
			`->(` + nodeName + `)`
		args[nodeName + `id`] = conso.Conso.ID
		args[relName + `amount`] = conso.Amount
		i++
	}

	if sumPrice != o.Price {
		return order, errors.New("CreateOrder (Price Check) : sum of price in users != order.Price")
	}

	/* Final */
	req = reqMatch + ` ` + reqWhere + ` ` + reqCreate + ` RETURN ID(o)`
	stmt, err := conn.PrepareNeo(req)
	if err != nil {
		fmt.Println("CreateOrder (PrepareNeo) : " + err.Error())
		return order, err
	}
	defer stmt.Close()

	rows, err := stmt.QueryNeo(args)
	if err != nil {
		fmt.Println("CreateOrder (QueryNeo) : " + err.Error())
		return order, err
	}

	data, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("CreateOrder (NextNeo) : " + err.Error())
		return order, err
	}
	orderID := data[0].(int64)
	
	order, err = s.GetOrder(ctx, orderID)
	if err != nil {
		fmt.Println("CreateOrder (GetOrder) : " + err.Error())
		return order, err
	}
	return order, err
}

/*************** Endpoint ***************/
type createOrderRequest struct {
	Order	Order `json:"order"`
}

type createOrderResponse struct {
	Order	Order  `json:"order"`
	Err		string `json:"err,omitempty"`
}

func CreateOrderEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createOrderRequest)
		order, err := svc.CreateOrder(ctx, req.Order)

		// Create node
		if err != nil {
			fmt.Println("Error CreateOrderEndpoint 1 : ", err.Error())
			return createOrderResponse{Order: order, Err: err.Error()}, nil
		}

		return createOrderResponse{Order: order, Err: ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPCreateOrderRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		fmt.Println("Error DecodeHTTPCreateOrderRequest : ", err.Error(), err)
		return nil, err
	}
	return request, nil
}

func DecodeHTTPCreateOrderResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response createOrderResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		fmt.Println("Error DecodeHTTPCreateOrderResponse : ", err.Error())
		return nil, err
	}
	return response, nil
}

func CreateOrderHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/orders").Handler(httptransport.NewServer(
		endpoints.CreateOrderEndpoint,
		DecodeHTTPCreateOrderRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "CreateOrder", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) CreateOrder(ctx context.Context, o Order) (Order, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "createOrder",
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.CreateOrder(ctx, o)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) CreateOrder(ctx context.Context, o Order) (Order, error) {
	v, err := mw.next.CreateOrder(ctx, o)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildCreateOrderEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "CreateOrder")
		csLogger := log.With(logger, "method", "CreateOrder")

		csEndpoint = CreateOrderEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "CreateOrder")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
func (e Endpoints) CreateOrder(ctx context.Context, o Order) (Order, error) {
	request := createOrderRequest{Order: o}
	response, err := e.CreateOrderEndpoint(ctx, request)
	if err != nil {
		fmt.Println("Error CreateOrder : ", err.Error())
		return o, err
	}
	return response.(createOrderResponse).Order, str2err(response.(createOrderResponse).Err)
}

func ClientCreateOrder(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, "/orders"),
		EncodeHTTPGenericRequest,
		DecodeHTTPCreateOrderResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "CreateOrder")(ceEndpoint)
	return ceEndpoint, nil
}
