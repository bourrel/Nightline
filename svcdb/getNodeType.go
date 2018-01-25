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
	"github.com/johnnadratowski/golang-neo4j-bolt-driver/structures/graph"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	httptransport "github.com/go-kit/kit/transport/http"
)

/*************** Service ***************/
func (s Service) GetNodeType(_ context.Context, nodeID int64) (string, error) {
	var nodeType string

	conn, err := WaitConnection(5)
	if err != nil {
		fmt.Println("getNodeType (WaitConnection) : " + err.Error())
		return "", err
	}
	defer CloseConnection(conn)

	stmt, err := conn.PrepareNeo(`
		MATCH (node)
		WHERE ID(node) = {id}
		return node
	`)
	defer stmt.Close()
	if err != nil {
		fmt.Println("GetNodeType (PrepareNeo) : " + err.Error())
		return nodeType, err
	}

	rows, err := stmt.QueryNeo(map[string]interface{}{
		"id": nodeID,
	})

	if err != nil {
		fmt.Println("GetNodeType (QueryNeo) : " + err.Error())
		return nodeType, err
	}

	row, _, err := rows.NextNeo()
	if err != nil {
		fmt.Println("GetNodeType (NextNeo) : " + err.Error())
		return nodeType, err
	}

	nodeType = row[0].(graph.Node).Labels[0]

	return nodeType, nil
}

/*************** Endpoint ***************/
type getNodeTypeRequest struct {
	NodeID int64 `json:"id"`
}

type getNodeTypeResponse struct {
	Consos string `json:"nodeType"`
	Err    string `json:"err,omitempty"`
}

func GetNodeTypeEndpoint(svc IService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getNodeTypeRequest)
		nodeType, err := svc.GetNodeType(ctx, req.NodeID)
		if err != nil {
			return getNodeTypeResponse{Consos: nodeType, Err: err.Error()}, nil
		}
		return getNodeTypeResponse{Consos: nodeType, Err: ""}, nil
	}
}

/*************** Transport ***************/
func DecodeHTTPGetNodeTypeRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request getNodeTypeRequest

	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	nodeID, err := strconv.ParseInt(mux.Vars(r)["NodeID"], 10, 64)
	if err != nil {
		return nil, err
	}
	(&request).NodeID = nodeID

	return request, nil
}

func DecodeHTTPGetNodeTypeResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var response getNodeTypeResponse
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response, nil
}

func GetNodeTypeHTTPHandler(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger, r *mux.Router, options []httptransport.ServerOption) *mux.Route {
	route := r.Methods("GET").Path("/node_type/{NodeID}").Handler(httptransport.NewServer(
		endpoints.GetNodeTypeEndpoint,
		DecodeHTTPGetNodeTypeRequest,
		EncodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(opentracing.HTTPToContext(tracer, "GetNodeType", logger)))...,
	))
	return route
}

/*************** Logger ***************/
/* Logger */
func (mw serviceLoggingMiddleware) GetNodeType(ctx context.Context, nodeID int64) (string, error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "getNodeType",
			"nodeID", nodeID,
			"took", time.Since(begin),
		)
	}(time.Now())
	return mw.next.GetNodeType(ctx, nodeID)
}

/*************** Instrumenting ***************/
/* Instrumenting */
func (mw serviceInstrumentingMiddleware) GetNodeType(ctx context.Context, nodeID int64) (string, error) {
	v, err := mw.next.GetNodeType(ctx, nodeID)
	mw.ints.Add(1)
	return v, err
}

/*************** Main ***************/
/* Main */
func BuildGetNodeTypeEndpoint(svc IService, logger log.Logger, tracer stdopentracing.Tracer, duration metrics.Histogram) endpoint.Endpoint {
	var gefmEndpoint endpoint.Endpoint
	{
		gefmDuration := duration.With("method", "GetNodeType")
		gefmLogger := log.With(logger, "method", "GetNodeType")

		gefmEndpoint = GetNodeTypeEndpoint(svc)
		gefmEndpoint = opentracing.TraceServer(tracer, "GetNodeType")(gefmEndpoint)
		gefmEndpoint = EndpointLoggingMiddleware(gefmLogger)(gefmEndpoint)
		gefmEndpoint = EndpointInstrumentingMiddleware(gefmDuration)(gefmEndpoint)
	}
	return gefmEndpoint
}

/*************** Client ***************/
/* Client */
// Forget limiter & circuitbreaker for now kthx
func (e Endpoints) GetNodeType(ctx context.Context, nodeID int64) (string, error) {
	var s string

	request := getNodeTypeRequest{NodeID: nodeID}
	response, err := e.GetNodeTypeEndpoint(ctx, request)
	if err != nil {
		return s, err
	}
	s = response.(getNodeTypeResponse).Consos
	return s, str2err(response.(getNodeTypeResponse).Err)
}

func EncodeHTTPGetNodeTypeRequest(ctx context.Context, r *http.Request, request interface{}) error {
	route := mux.NewRouter()
	id := fmt.Sprintf("%v", request.(getNodeTypeRequest).NodeID)
	encodedUrl, err := route.Path(r.URL.Path).URL("NodeID", id)
	if err != nil {
		return err
	}
	r.URL.Path = encodedUrl.Path
	return nil
}

func ClientGetNodeType(u *url.URL, logger log.Logger, tracer stdopentracing.Tracer) (endpoint.Endpoint, error) {
	var gefmEndpoint endpoint.Endpoint

	gefmEndpoint = httptransport.NewClient(
		"GET",
		copyURL(u, "/node_type/{NodeID}"),
		EncodeHTTPGetNodeTypeRequest,
		DecodeHTTPGetNodeTypeResponse,
		httptransport.ClientBefore(opentracing.ContextToHTTP(tracer, logger)),
	).Endpoint()
	gefmEndpoint = opentracing.TraceClient(tracer, "GetNodeType")(gefmEndpoint)
	return gefmEndpoint, nil
}
