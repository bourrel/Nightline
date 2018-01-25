package svcdb

import (
	"context"
	"encoding/json"
	"io"
	"fmt"
	"net/http"
	"net/url"

	"strconv"
	"time"

	"github.com/go-kit/kit/tracing/opentracing"
	mux "github.com/gorilla/mux"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/johnnadratowski/golang-neo4j-bolt-driver/structures/graph"
)

/*************** Service ***************/
func (s Service) GetOrder(_ context.Context, orderID int64) (Order, error) {
	var elementsNode map[int64]graph.Node
	var elementsRel []graph.Relationship
	var order Order

	elementsNode = make(map[int64]graph.Node)
	elementsDone := make(map[int64]bool)
	
	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetOrder (WaitConnection) : " + err.Error())
		return order, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
        MATCH (o:ORDER) WHERE ID(o) = {id} WITH o
		MATCH
        (o)-[ur:TO]->(u:USER),
        (o)-[cr:FOR]->(c:CONSO),
        (o)-[sor:DURING]->(so:SOIREE)
        OPTIONAL MATCH (o)-[str:DONE]->(st:STEP)
        RETURN o, u, ur, c, cr, so, sor, st, str
    `)
	defer stmt.Close()

	if err != nil {
		fmt.Println("GetOrder (PrepareNeo) : " + err.Error())
		return order, err
	}

	rowss, err := stmt.QueryNeo(map[string]interface{}{
		"id": orderID,
	})

	if err != nil {
		fmt.Println("GetOrder (QueryNeo) : " + err.Error())
		return order, err
	}

	rows, _, err := rowss.NextNeo()
	for rows != nil && err == nil {
		if err != nil && err != io.EOF {
			fmt.Println("GetOrder (NextNeo) : " + err.Error())
			return order, err
		} else if err != io.EOF {
			for _, row := range rows {
				switch row.(type) {
				case graph.Node:
					elementsNode[row.(graph.Node).NodeIdentity] = row.(graph.Node)
					if row.(graph.Node).Labels[0] == "ORDER" {
						(&order).NodeToOrder(row.(graph.Node))
					}
				case graph.Relationship:
					elementsRel = append(elementsRel, row.(graph.Relationship))
				}
			}

			for _, rel := range elementsRel {
				orderNode := elementsNode[rel.StartNodeIdentity]
				targetNode := elementsNode[rel.EndNodeIdentity]
				if elementsDone[rel.EndNodeIdentity] == false {
					switch targetNode.Labels[0] {
					case "SOIREE":
						(&order).RelationToSoiree(orderNode, rel, targetNode)
					case "USER":
						(&order).RelationAddUser(orderNode, rel, targetNode)
					case "CONSO":
						(&order).RelationAddConso(orderNode, rel, targetNode)
					case "STEP":
						(&order).RelationAddStep(orderNode, rel, targetNode)
					}
					elementsDone[rel.EndNodeIdentity] = true
				}
			}
		}
		rows, _, err = rowss.NextNeo()
	}

	return order, nil
}

/*************** Endpoint ***************/
type getOrderRequest struct {
	OrderID int64 `json:"id"`
}

type getOrderResponse struct {
	Order Order  `json:"order"`
	Err   string `json:"err,omitempty"`
}

func GetOrderEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getOrderRequest)
		order, err := svc.GetOrder(ctx, req.OrderID)
		if err != nil {
			return getOrderResponse{order, err.Error()}, nil
		}
		return getOrderResponse{order, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetOrderRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getOrderRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	orderID, err := strconv.ParseInt(mux.Vars(r)["OrderID"], 10, 64)
	if err != nil {
		return nil, err
	}
	(&request).OrderID = orderID

	return request, nil
}

func DecodeHTTPGetOrderResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getOrderResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetOrderHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/orders/{OrderID:[0-9]+}").Handler(httptransport.NewServer(
		endpoints.GetOrderEndpoint,
		DecodeHTTPGetOrderRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetOrder", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetOrder(ctx context.Context, orderID int64) (Order, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getOrder",
			"orderID", orderID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetOrder(ctx, orderID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetOrder(ctx context.Context, orderID int64) (Order, error) {
	v, err := mw.next.GetOrder(ctx, orderID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetOrderEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetOrder")
		csLogger := log.With(logger, "method", "GetOrder")

		csEndpoint = GetOrderEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetOrder")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetOrder(ctx context.Context, orderID int64) (Order, error) {
	var order Order

	request := getOrderRequest{OrderID: orderID}
	response, err := e.GetOrderEndpoint(ctx, request)
	if err != nil {
		return order, err
	}
	order = response.(getOrderResponse).Order
	return order, str2err(response.(getOrderResponse).Err)
}

func EncodeHTTPGetOrderRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(getOrderRequest).OrderID)
	encodedUrl, err := route.Path(r.URL.Path).URL("OrderID", id)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetOrder(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/orders/{OrderID}"),
		EncodeHTTPGetOrderRequest,
		DecodeHTTPGetOrderResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "GetOrder")(ceEndpoint)
	return ceEndpoint, nil
}
