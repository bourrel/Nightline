package svcdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/johnnadratowski/golang-neo4j-bolt-driver"

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
func (s Service) CreateMessage(c context.Context, nodeType string, m Message) (Message, error) {
	var message Message
	var stmt golangNeo4jBoltDriver.Stmt

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("CreateMessage (WaitConnection) : " + err.Error())
		return message, err
	}
	defer CloseConnection(conn)

	fmt.Println(nodeType)
	fmt.Println(m.From)
	fmt.Println(m.To)
	fmt.Println(m.Text)

	if nodeType == "USER" {
		stmt, err = conn.PrepareNeo(`
			MATCH (u:USER) WHERE ID(u) = {initID}
			MATCH (f:USER) WHERE ID(f) = {friendID}
			CREATE (f)-[:FROM]->(c:MESSAGE {
				Text: {message},
				Date: {time}
			} )-[:TO]->(u)
			RETURN ID(u), c, ID(f)
		`)
	} else if nodeType == "GROUP" {
		stmt, err = conn.PrepareNeo(`
			MATCH (u:USER) WHERE ID(u) = {initID}
			MATCH (g:GROUP) WHERE ID(g) = {friendID}
			CREATE (u)-[:FROM]->(c:MESSAGE {
				Text: {message},
				Date: {time}
			} )-[:TO]->(g)
			RETURN ID(u), c, ID(g)
		`)
	} else {
		return message, errors.New("You can't send a message to this ")
	}

	defer stmt.Close()

	if err != nil {
		fmt.Println("CreateMessage (PrepareNeo) : " + err.Error())
		return message, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"initID":   m.From,
		"friendID": m.To,
		"message":  m.Text,
		"time":     time.Now().Format(timeForm),
	})

	if err != nil {
		fmt.Println("CreateMessage (QueryNeo) : " + err.Error())
		return message, err
	}

	data, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("CreateMessage (NextNeo) : " + err.Error())
		return message, err
	}

	(&message).NodeToMessage(data[1].(graph.Node))
	message.From = data[0].(int64)
	message.To = data[2].(int64)

	return message, err
}

/*************** Endpoint ***************/
type createMessageRequest struct {
	Message Message `json:"message"`
}

type createMessageResponse struct {
	Message Message `json:"message"`
	Err     string  `json:"err,omitempty"`
}

func CreateMessageEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createMessageRequest)

		nType, err := svc.GetNodeType(ctx, req.Message.To)
		if err != nil {
			fmt.Println("CreateMessageEndpoint 1 : " + err.Error())
			return createMessageResponse{Err: err.Error()}, nil
		}

		message, err := svc.CreateMessage(ctx, nType, req.Message)
		if err != nil {
			fmt.Println("CreateMessageEndpoint 2 : " + err.Error())
			return createMessageResponse{Err: err.Error()}, nil
		}

		return createMessageResponse{message, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPCreateMessageRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request createMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func DecodeHTTPCreateMessageResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response createMessageResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func CreateMessageHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("POST").Path("/messages").Handler(httptransport.NewServer(
		endpoints.CreateMessageEndpoint,
		DecodeHTTPCreateMessageRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "CreateMessage", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) CreateMessage(ctx context.Context, nType string, m Message) (Message, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "createMessage",
			"initiator", m.From,
			"recipient", strconv.FormatInt(m.To, 10)+" ("+nType+")",
			"m", m.Text,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.CreateMessage(ctx, nType, m)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) CreateMessage(ctx context.Context, nType string, m Message) (Message, error) {
	v, err := mw.next.CreateMessage(ctx, nType, m)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildCreateMessageEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "CreateMessage")
		csLogger := log.With(logger, "method", "CreateMessage")

		csEndpoint = CreateMessageEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "CreateMessage")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
func (e Endpoints) CreateMessage(ctx context.Context, nType string, m Message) (Message, error) {
	request := createMessageRequest{Message: m}
	response, err := e.CreateMessageEndpoint(ctx, request)
	if err != nil {
		return m, err
	}
	return response.(createMessageResponse).Message, str2err(response.(createMessageResponse).Err)
}

func ClientCreateMessage(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"POST",
		copyURL(u, "/messages"),
		EncodeHTTPGenericRequest,
		DecodeHTTPCreateMessageResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "CreateMessage")(ceEndpoint)
	return ceEndpoint, nil
}
