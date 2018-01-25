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
func (s Service) GetLastMessages(_ context.Context, recipient, initiator int64) ([]Message, error) {
	var messages []Message
	var tmpMessage Message

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetLastMessages (WaitConnection) : " + err.Error())
		return nil, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (user)-[a]-(m:MESSAGE)-[b]-(recipient)
		WHERE
			ID(user) = {initiator} AND
			ID(recipient) = {recipient}
		RETURN user, a, m, b, recipient
	`)
	defer stmt.Close()
	if err != nil {
		fmt.Println("GetLastMessages (PrepareNeo) : " + err.Error())
		return messages, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"recipient": recipient,
		"initiator": initiator,
	})

	if err != nil {
		fmt.Println("GetLastMessages (QueryNeo) : " + err.Error())
		return messages, err
	}

	row, _, err := rows.NextNeo()
	for row != nil && err == nil {
		if err != nil && err != io.EOF {
			fmt.Println("GetLastMessages (---) : " + err.Error())
			panic(err)
		} else if err != io.EOF {
			(&tmpMessage).NodeToMessage(row[2].(graph.Node))

			if row[1].(graph.Relationship).Type == "FROM" {
				tmpMessage.From = row[0].(graph.Node).NodeIdentity
				tmpMessage.To = row[4].(graph.Node).NodeIdentity
			} else {
				tmpMessage.From = row[4].(graph.Node).NodeIdentity
				tmpMessage.To = row[0].(graph.Node).NodeIdentity
			}

			messages = append(messages, tmpMessage)
		}
		row, _, err = rows.NextNeo()
	}
	return messages, nil
}

/*************** Endpoint ***************/
type getLastMessagesRequest struct {
	recipient int64 `json:"recipient"`
	initiator int64 `json:"initiator"`
}

type getLastMessagesResponse struct {
	Messages []Message `json:"messages"`
	Err      string    `json:"err,omitempty"`
}

func GetLastMessagesEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getLastMessagesRequest)
		messages, err := svc.GetLastMessages(ctx, req.recipient, req.initiator)
		if err != nil {
			return getLastMessagesResponse{Messages: messages, Err: err.Error()}, nil
		}
		return getLastMessagesResponse{Messages: messages, Err: ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetLastMessagesRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getLastMessagesRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	from, err := strconv.ParseInt(mux.Vars(r)["from"], 10, 64)
	if err != nil {
		return nil, err
	}
	(&request).initiator = from

	to, err := strconv.ParseInt(mux.Vars(r)["to"], 10, 64)
	if err != nil {
		return nil, err
	}
	(&request).recipient = to

	return request, nil
}

func DecodeHTTPGetLastMessagesResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getLastMessagesResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetLastMessagesHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/messages/{from}/{to}").Handler(httptransport.NewServer(
		endpoints.GetLastMessagesEndpoint,
		DecodeHTTPGetLastMessagesRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetLastMessages", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetLastMessages(ctx context.Context, recipient, initiator int64) ([]Message, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getLastMessages",
			"recipient", recipient,
			"initiator", initiator,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetLastMessages(ctx, recipient, initiator)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetLastMessages(ctx context.Context, recipient, initiator int64) ([]Message, error) {
	v, err := mw.next.GetLastMessages(ctx, recipient, initiator)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetLastMessagesEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var gefmEndpoint endpoint.Endpoint
	{
		gefmDuration := duration.With("method", "GetLastMessages")
		gefmLogger := log.With(logger, "method", "GetLastMessages")

		gefmEndpoint = GetLastMessagesEndpoint(svc)
		gefmEndpoint = opentracing.TraceServer(tracer, "GetLastMessages")(gefmEndpoint)
		gefmEndpoint = EndpointLoggingMiddleware(gefmLogger)(gefmEndpoint)
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetLastMessages(ctx context.Context, recipient, initiator int64) ([]Message, error) {
	var s []Message

	request := getLastMessagesRequest{recipient, initiator}
	response, err := e.GetLastMessagesEndpoint(ctx, request)
	if err != nil {
		return s, err
	}
	s = response.(getLastMessagesResponse).Messages
	return s, str2err(response.(getLastMessagesResponse).Err)
}

func EncodeHTTPGetLastMessagesRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()

	to := fmt.Sprintf("%v", request.(getLastMessagesRequest).recipient)
	from := fmt.Sprintf("%v", request.(getLastMessagesRequest).initiator)

	encodedUrl, err := route.Path(r.URL.Path).URL("from", from, "to", to)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path

	return nil
}

func ClientGetLastMessages(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/messages/{from}/{to}"),
		EncodeHTTPGetLastMessagesRequest,
		DecodeHTTPGetLastMessagesResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "GetLastMessages")(gefmEndpoint)
	return gefmEndpoint, nil
}
