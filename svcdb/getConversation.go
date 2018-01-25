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
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"
)

/*************** Service ***************/
func (s Service) GetConversationByID(ctx context.Context, convID int64) (Conversation, error) {
	var conv Conversation

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("GetConversationByID (WaitConnection) : " + err.Error())
		return conv, err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (u:USER)-[]->(c:CONVERSATION)<-[]-(f)
		OPTIONAL MATCH (c)-[]-(m:MESSAGE)
		WHERE ID(c) = 177
		RETURN ID(c), ID(f), COUNT(m)
	`)
	defer stmt.Close()

	if err != nil {
		fmt.Println("GetConversationByID (PrepareNeo) : " + err.Error())
		return conv, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": convID,
	})

	if err != nil {
		fmt.Println("GetConversationByID (QueryNeo) : " + err.Error())
		return conv, err
	}

	row, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("GetConversationByID (NextNeo) : " + err.Error())
		return conv, err
	}

	(&conv).NodeToConversation(row)

	if conv.RecipientID > 0 {
		conv.RecipientType, err = s.GetNodeType(ctx, conv.RecipientID)
		if err != nil {
			fmt.Println("GetConversationByID (Recipient Type) : " + err.Error())
			return conv, err
		}
	}

	return conv, nil
}

/*************** Endpoint ***************/
type getConversationByIDRequest struct {
	ConversationID int64 `json:"id"`
}

type getConversationByIDResponse struct {
	Conversation Conversation `json:"conversation"`
	Err          string       `json:"err,omitempty"`
}

func GetConversationByIDEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getConversationByIDRequest)

		conversation, err := svc.GetConversationByID(ctx, req.ConversationID)
		if err != nil {
			fmt.Println("Error GetConversationByIDEndpoint 1 : ", err.Error())
			return getConversationByIDResponse{conversation, err.Error()}, nil
		}

		return getConversationByIDResponse{conversation, ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetConversationByIDRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getConversationByIDRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	conversationID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		return nil, err
	}
	(&request).ConversationID = conversationID

	return request, nil
}

func DecodeHTTPGetConversationByIDResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getConversationByIDResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetConversationByIDHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/conversations/{id}").Handler(httptransport.NewServer(
		endpoints.GetConversationByIDEndpoint,
		DecodeHTTPGetConversationByIDRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetConversationByID", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetConversationByID(ctx context.Context, conversationID int64) (Conversation, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getConversationByID",
			"conversationID", conversationID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetConversationByID(ctx, conversationID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetConversationByID(ctx context.Context, conversationID int64) (Conversation, error) {
	v, err := mw.next.GetConversationByID(ctx, conversationID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetConversationByIDEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var csEndpoint endpoint.Endpoint
	{
		csDuration := duration.With("method", "GetConversationByID")
		csLogger := log.With(logger, "method", "GetConversationByID")

		csEndpoint = GetConversationByIDEndpoint(svc)
		csEndpoint = opentracing.TraceServer(tracer, "GetConversationByID")(csEndpoint)
		csEndpoint = EndpointLoggingMiddleware(csLogger)(csEndpoint)
		csEndpoint = EndpointInstrumentingMiddleware(csDuration)(csEndpoint)
	}
	return csEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetConversationByID(ctx context.Context, etID int64) (Conversation, error) {
	var et Conversation

	request := getConversationByIDRequest{ConversationID: etID}
	response, err := e.GetConversationByIDEndpoint(ctx, request)
	if err != nil {
		return et, err
	}
	et = response.(getConversationByIDResponse).Conversation
	return et, str2err(response.(getConversationByIDResponse).Err)
}

func EncodeHTTPGetConversationByIDRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(getNodeTypeRequest).NodeID)
	encodedUrl, err := route.Path(r.URL.Path).URL("id", id)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetConversationByID(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var ceEndpoint endpoint.Endpoint

	ceEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/conversations/{id}"),
		EncodeHTTPGetConversationByIDRequest,
		DecodeHTTPGetConversationByIDResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	ceEndpoint = opentracing.TraceClient(tracer, "GetConversationByID")(ceEndpoint)
	return ceEndpoint, nil
}
